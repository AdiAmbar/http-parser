package request

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"http.protocol/internal/headers"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Headers     *headers.Headers
	State       ParseState
	Body        string
}

func getInt(headers *headers.Headers, fieldName string, defaultValue int) int {
	valueStr, exist := headers.Get(fieldName)

	if !exist {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func (r *Request) hasBody() bool {
	// TODO: Whendoing chuncked coding, this method will need to be updated
	length := getInt(r.Headers, "content-length", 0)
	return length > 0
}

type ParseState string

const (
	StateInit    ParseState = "init"
	StateDone    ParseState = "done"
	StateError   ParseState = "error"
	StateHeaders ParseState = "headers"
	StateBody    ParseState = "body"
)

const bufferSize = 1024

var ERROR_INCOMPLETE_DATA = fmt.Errorf("incomplete data")
var ERROR_BAD_START_LINE = fmt.Errorf("bad start line")
var ERROR_UNSUPPORTED_HTTP_VERSION = fmt.Errorf("unsupported HTTP version")
var ERROR_REQUEST_IN_ERROR_STATE = fmt.Errorf("request in error state")
var SEPARATOR_CRLF = []byte("\r\n")

func newRequest() *Request {
	return &Request{
		State:   StateInit,
		Headers: headers.NewHeaders(),
	}
}

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, SEPARATOR_CRLF)
	if idx == -1 {
		return nil, 0, nil
		// return nil, 0, ERROR_INCOMPLETE_DATA
	}

	startLine := b[:idx]
	// restOfMsg := b[idx+len(SEPARATOR_CRLF):]
	read := idx + len(SEPARATOR_CRLF)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ERROR_BAD_START_LINE
	}

	httpParts := bytes.Split(parts[2], []byte("/"))
	if len(httpParts) != 2 || string(httpParts[0]) != "HTTP" || string(httpParts[1]) != "1.1" {
		return nil, 0, ERROR_UNSUPPORTED_HTTP_VERSION
	}

	rl := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpParts[1]),
	}

	return rl, read, nil
}

func (r *Request) parse(data []byte) (int, error) {

	read := 0

outer:
	for {
		currentData := data[read:]
		if len(currentData) == 0 {
			break outer
		}

		switch r.State {
		case StateError:
			return 0, ERROR_REQUEST_IN_ERROR_STATE

		case StateInit:
			rl, n, err := parseRequestLine(currentData)
			if err != nil {
				r.State = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}
			r.RequestLine = *rl
			read += n

			r.State = StateHeaders

		case StateHeaders:
			n, done, err := r.Headers.Parse(currentData)
			if err != nil {
				r.State = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}

			read += n

			if done {
				if r.hasBody() {
					r.State = StateBody
				} else {
					r.State = StateDone
				}
			}

		case StateBody:
			length := getInt(r.Headers, "content-length", 0)
			if length == 0 {
				panic("Chunked not implemented yet")
			}

			remaining := min(length-len(r.Body), len(currentData))
			r.Body += string(currentData[:remaining])
			read += remaining

			if len(r.Body) == length {
				r.State = StateDone
			}

		case StateDone:
			break outer

		default:
			panic("something went wrong in parse function")
		}
	}

	return read, nil
}

func (r *Request) done() bool {
	return r.State == StateDone || r.State == StateError
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()

	// TODO: buffer size of 1024 is arbitrary, we should consider how to handle larger requests that may not fit in the buffer
	buf := make([]byte, bufferSize)
	bufLen := 0

	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		if err != nil {
			return nil, err
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[readN:bufLen])
		bufLen -= readN
	}

	request.RequestLine.Method = string(bytes.ToUpper([]byte(request.RequestLine.Method)))

	return request, nil
}
