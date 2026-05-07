package request

import (
	"bytes"
	"fmt"
	"io"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	State       ParseState
}

type ParseState string

const (
	StateInit  ParseState = "init"
	StateDone  ParseState = "done"
	StateError ParseState = "error"
)

const bufferSize = 1024

var ERROR_INCOMPLETE_DATA = fmt.Errorf("incomplete data")
var ERROR_BAD_START_LINE = fmt.Errorf("bad start line")
var ERROR_UNSUPPORTED_HTTP_VERSION = fmt.Errorf("unsupported HTTP version")
var ERROR_REQUEST_IN_ERROR_STATE = fmt.Errorf("request in error state")
var SEPARATOR_CRLF = []byte("\r\n")

func newRequest() *Request {
	return &Request{
		State: StateInit,
	}
}

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, SEPARATOR_CRLF)
	if idx == -1 {
		return nil, 0, ERROR_INCOMPLETE_DATA
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
		switch r.State {
		case StateError:
			return 0, ERROR_REQUEST_IN_ERROR_STATE

		case StateInit:
			rl, n, err := parseRequestLine(data[read:])
			if err != nil {
				r.State = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}
			r.RequestLine = *rl
			read += n

			r.State = StateDone

		case StateDone:
			break outer
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

	return request, nil
}
