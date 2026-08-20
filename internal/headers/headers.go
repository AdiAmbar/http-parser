package headers

import (
	"bytes"
	"fmt"
	"strings"
)

var crlf = []byte("\r\n")

var ERROR_BAD_HEADER_LINE = fmt.Errorf("bad header line")
var ERROR_MALFORMED_FIELD_NAME = fmt.Errorf("malformed field name")
var ERROR_MALFORMED_HEADER_NAME = fmt.Errorf("malformed header name")
var ERROR_HEADER_KEY_CONTAINS_INVALID_CARACTERS = fmt.Errorf("header key contains invalid characters")

type Headers struct {
	headers map[string]string
}

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}

func (h *Headers) Get(name string) (string, bool) {
	str, ok := h.headers[strings.ToLower(name)]
	return str, ok
}

func (h *Headers) Replace(name, value string) {
	name = strings.ToLower(name)
	h.headers[name] = value
}

func (h *Headers) Delete(name string) {
	name = strings.ToLower(name)
	delete(h.headers, name)
}

func (h *Headers) Set(name, value string) {
	name = strings.ToLower(name)

	if v, ok := h.headers[name]; ok {
		h.headers[name] = fmt.Sprintf("%s, %s", v, value)
	} else {
		h.headers[name] = value
	}
}

func isValidToken(str []byte) bool {
	for _, ch := range str {
		found := false
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			found = true
		}

		switch ch {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			found = true
		}

		if !found {
			return false
		}
	}

	return true
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

	fieldNameLower := strings.ToLower(string(fieldName))

	return fieldNameLower, string(fieldValue), nil
}

func (h *Headers) ForEach(cb func(name, value string)) {
	for name, value := range h.headers {
		cb(name, value)
	}
}

func (h *Headers) Parse(data []byte) (int, bool, error) {

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

		fieldName, fieldValue, err := parseHeaderLine(data[read : read+idx])
		if err != nil {
			return 0, false, err
		}

		if !isValidToken([]byte(fieldName)) {
			return 0, false, ERROR_MALFORMED_HEADER_NAME
		}

		read += idx + len(crlf)
		h.Set(fieldName, fieldValue)
	}

	return read, done, nil
}
