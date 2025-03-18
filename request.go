package custyhttp

import (
	"fmt"
	"strings"
	"sync"
)

type Request struct {
	Method        string
	body          []byte
	RequestTarget string
	Path          string
	Protocol      string
	headers       []string
	Host          string
	UserAgent     string
	Accept        string
	ContentType   string
	ContentLength int
	queryParams   map[string]string
	mu            sync.RWMutex
}

func (r *Request) String() string {
	headers := strings.Join(r.Headers(), "\r\n")

	req := fmt.Sprintf("%s %s %s\r\n%s\r\n", r.Method, r.RequestTarget, r.Protocol, headers)

	if len(r.Body()) == 0 {
		return req + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", req, string(r.Body()))
}

func (r *Request) Body() []byte {
	r.mu.Lock()
	body := make([]byte, len(r.body))
	copy(body, r.body)
	r.mu.Unlock()

	return body
}

func (r *Request) Headers() []string {
	r.mu.Lock()
	headers := make([]string, len(r.headers))
	copy(headers, r.headers)
	r.mu.Unlock()

	return headers
}

func (r *Request) QueryParams() map[string]string {
	return r.queryParams
}

func (r *Request) setHeaders(headers []string) {
	r.mu.Lock()
	r.headers = headers
	r.mu.Unlock()
}

func (r *Request) WriteBytesToBody(b []byte) {
	r.mu.Lock()
	r.body = append(r.body, b...)
	r.mu.Unlock()
}
