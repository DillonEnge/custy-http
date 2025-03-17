package custyhttp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type HandlerFunc func(ctx context.Context, r *Request) (*Response, error)

type Request struct {
	Method        string
	Body          []byte
	RequestTarget string
	Path          string
	Protocol      string
	Headers       []string
	Host          string
	UserAgent     string
	Accept        string
	ContentType   string
	ContentLength int
	Raw           []byte
	queryParams   map[string]string
}

type Response struct {
	Body          []byte
	StatusCode    int
	StatusText    string
	Protocol      string
	Headers       []string
	Server        string
	Date          string
	CacheControl  string
	ContentType   string
	ContentLength int
	Connection    string
	ETag          string
	LastModified  string
	mu            sync.Mutex
}

func BadRequest() *Response {
	return &Response{
		StatusCode: 400,
		StatusText: "Bad Request",
	}
}

func InternalServerError() *Response {
	return &Response{
		StatusCode: 500,
		StatusText: "Internal Server Error",
	}
}

func NotImplemented() *Response {
	return &Response{
		StatusCode: 501,
		StatusText: "Not Implemented",
	}
}

func NotFound() *Response {
	return &Response{
		StatusCode: 404,
		StatusText: "Not Found",
	}
}

func NoContent() *Response {
	return &Response{
		StatusCode: 204,
		StatusText: "No Content",
	}
}

func (r Request) String() string {
	headers := strings.Join(r.Headers, "\r\n")

	req := fmt.Sprintf("%s %s %s\r\n%s\r\n", r.Method, r.RequestTarget, r.Protocol, headers)

	if len(r.Body) == 0 {
		return req + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", req, string(r.Body))
}

func (r *Response) String() string {
	headers := strings.Join(r.Headers, "\r\n")

	resp := fmt.Sprintf("%s %d %s\r\n%s\r\n", r.Protocol, r.StatusCode, r.StatusText, headers)

	if len(r.Body) == 0 {
		return resp + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", resp, string(r.Body))
}

func (r *Request) QueryParams() map[string]string {
	return r.queryParams
}

func (r *Response) populateHeaders() {
	if r.Server != "" {
		r.addHeader("Server", r.Server)
	}
	if r.Date != "" {
		r.addHeader("Date", r.Date)
	}
	if r.CacheControl != "" {
		r.addHeader("Cache-Control", r.CacheControl)
	}
	if r.ContentType != "" {
		r.addHeader("Content-Type", r.ContentType)
	}
	if r.ContentLength != 0 {
		r.addHeader("Content-Length", strconv.Itoa(r.ContentLength))
	}
	if r.Connection != "" {
		r.addHeader("Connection", r.Connection)
	}
	if r.ETag != "" {
		r.addHeader("ETag", r.ETag)
	}
	if r.LastModified != "" {
		r.addHeader("Last-Modified", r.LastModified)
	}
}

func (r *Response) addHeader(key, value string) {
	r.mu.Lock()
	r.Headers = append(r.Headers, fmt.Sprintf("%s: %s", key, value))
	r.mu.Unlock()
}
