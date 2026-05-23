package headers

import (
	"bytes"
	"fmt"
)

type Headers map[string]string

var crlf = []byte("\r\n")

var ERROR_BAD_HEADER_LINE = fmt.Errorf("bad header line")
var ERROR_MALFORMED_FIELD_NAME = fmt.Errorf("malformed field name")

func NewHeaders() Headers {
	return make(Headers)
}

func parseHeaderLine(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", ERROR_BAD_HEADER_LINE
	}

	fieldName := parts[0]
	fieldValue := bytes.TrimSpace(parts[1])

	if bytes.HasSuffix(fieldName, []byte(" ")) {
		return "", "", ERROR_MALFORMED_FIELD_NAME
	}

	return string(fieldName), string(fieldValue), nil
}

func (h Headers) Parse(data []byte) (int, bool, error) {

	read := 0
	done := false
	for {
		idx := bytes.Index(data[read:], crlf)
		if idx == -1 {
			break
		}

		if idx == 0 {
			done = true
			read += len(crlf)
			break
		}

		fieldName, fieldValue, err := parseHeaderLine(data[read:read+idx])
		if err != nil {
			return 0, false, err
		}

		read += idx + len(crlf)
		h[fieldName] = fieldValue
	}

	return read, done, nil
}
