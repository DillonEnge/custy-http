package custyhttp

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Method string

const (
	METHOD_HEAD    Method = "HEAD"
	METHOD_GET     Method = "GET"
	METHOD_POST    Method = "POST"
	METHOD_PATCH   Method = "PATCH"
	METHOD_PUT     Method = "PUT"
	METHOD_DELETE  Method = "DELETE"
	METHOD_OPTIONS Method = "OPTIONS"
)

type Server struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	routes       map[string]map[Method]HandlerFunc
	middlewares  []MiddlewareFunc
	mu           sync.RWMutex
}

func NewServer(opts ...func(*Server)) *Server {
	s := &Server{
		routes:      make(map[string]map[Method]HandlerFunc),
		middlewares: make([]MiddlewareFunc, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithReadTimeout(readTimeout time.Duration) func(s *Server) {
	return func(s *Server) {
		s.readTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) func(s *Server) {
	return func(s *Server) {
		s.writeTimeout = writeTimeout
	}
}

func (s *Server) ListenAndServe(ctx context.Context, address string) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		if s.readTimeout != 0 {
			conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		}
		if s.writeTimeout != 0 {
			conn.SetWriteDeadline(time.Now().Add(s.readTimeout))
		}

		go func() {
			resp, err := s.handleConn(ctx, conn)
			if err != nil {
				slog.Error("error handling connection", "err", err)
			}

			if resp.ContentLength == 0 && resp.Body() != nil && len(resp.Body()) > 0 {
				resp.ContentLength = len(resp.Body())
			}
			if resp.Protocol == "" {
				resp.Protocol = "HTTP/1.1"
			}
			if resp.Connection == "" {
				resp.Connection = "close"
			}

			resp.populateHeaders()

			if _, err := conn.Write([]byte(resp.String())); err != nil {
				slog.Error("failed to write to conn", "err", err)
			}
			if err := conn.Close(); err != nil {
				slog.Error("error closing conn", "err", err)
				return
			}
		}()
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) (*Response, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := s.parseRequest(conn)
	if err != nil {
		return badRequest(), err
	}

	var handler HandlerFunc

	s.mu.RLock()

	func() {
		handlers, ok := s.routes[req.Path]
		if !ok {
			handler = func(ctx context.Context, r *Request) (*Response, error) {
				return notFound(), fmt.Errorf("route not found")
			}
			return
		}

		handler, ok = handlers[Method(req.Method)]
		if !ok {
			handler = func(ctx context.Context, r *Request) (*Response, error) {
				return notImplemented(), fmt.Errorf("method %s not found for route %s", req.Method, req.RequestTarget)
			}
			return
		}
	}()

	mws := make([]MiddlewareFunc, len(s.middlewares))
	copy(mws, s.middlewares)

	s.mu.RUnlock()

	for i := len(mws); i > 0; i-- {
		handler = mws[i-1](handler)
	}

	resp, err := handler(ctx, req)
	if err != nil {
		return internalServerError(), err
	}
	if resp == nil {
		return internalServerError(), fmt.Errorf("nil response received")
	}

	if resp.StatusText == "" {
		switch resp.StatusCode {
		case 200:
			resp.StatusText = "OK"
		case 201:
			resp.StatusText = "Created"
		case 204:
			resp.StatusText = "No Content"
		case 400:
			resp.StatusText = "Bad Request"
		case 401:
			resp.StatusText = "Unauthorized"
		case 403:
			resp.StatusText = "Forbidden"
		case 404:
			resp.StatusText = "Not Found"
		case 418:
			resp.StatusText = "I'm a teapot"
		case 500:
			resp.StatusText = "Internal Server Error"
		case 501:
			resp.StatusText = "Not Implemented"
		}
	}

	return resp, nil
}

func (s *Server) registerRoute(method Method, route string, f HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.routes[route]
	if !ok {
		s.routes[route] = map[Method]HandlerFunc{
			method: f,
		}
		return
	}
	s.routes[route][method] = f
}

func (s *Server) registerMiddleware(f MiddlewareFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.middlewares = append(s.middlewares, f)
}

func (s *Server) parseRequest(conn net.Conn) (*Request, error) {
	crlfReplacer := strings.NewReplacer("\r", "", "\n", "")

	reader := bufio.NewReader(conn)

	startLine, err := reader.ReadString('\n')
	if err != nil {
		slog.Error("failed to read startLine", "err", err)
		return nil, err
	}
	startLineValues := strings.Split(crlfReplacer.Replace(startLine), " ")

	if len(startLineValues) != 3 {
		slog.Error("invalid start line", "startLine", startLine)
		return nil, err
	}

	req := Request{
		Method:        startLineValues[0],
		RequestTarget: startLineValues[1],
		Protocol:      startLineValues[2],
	}

	req.Path, req.queryParams = pathAndQueryParamsFromRequestTarget(req.RequestTarget)

	// Process headers
	headers := []string{}
	for {
		// t := scanner.Text()
		t, err := reader.ReadString('\n')
		if err != nil {
			slog.Error("failed to read header")
			return nil, err
		}

		if b, err := reader.Peek(2); string(b) == "\r\n" {
			if err != nil {
				slog.Error("error peeking for end of headers", "err", err)
				return nil, err
			}
			if _, err := reader.ReadBytes('\n'); err != nil {
				slog.Error("error clearing crlf sequence after headers", "err", err)
				return nil, err
			}
			break
		}

		headerSplit := strings.Split(t, ": ")

		if len(headerSplit) != 2 {
			slog.Error("invalid header", "header", t)
			return nil, fmt.Errorf("invalid header")
		}

		headerKey, headerValue := headerSplit[0], crlfReplacer.Replace(headerSplit[1])

		switch strings.ToLower(headerKey) {
		case "content-type":
			req.ContentType = headerValue
		case "content-length":
			req.ContentLength, _ = strconv.Atoi(headerValue)
		case "host":
			req.Host = headerValue
		case "user-agent":
			req.UserAgent = headerValue
		case "accept":
			req.Accept = headerValue
		}

		headers = append(headers, t)
	}

	req.setHeaders(headers)

	if req.ContentLength == 0 {
		slog.Info("no body expected", "req.ContentLength", req.ContentLength)

	}

	contentLength := 0

	for range req.ContentLength {
		// b := scanner.Bytes()
		b, err := reader.ReadByte()
		if err != nil {
			slog.Error("error reading next body byte", "err", err)
			return nil, err
		}
		contentLength++

		if contentLength > req.ContentLength {
			slog.Error("content length exceeds provided length", "contentLength", contentLength)
			return nil, fmt.Errorf("content length exceeds provided length")
		}

		req.writeBytesToBody([]byte{b})

		if contentLength == req.ContentLength {
			break
		}
	}

	if contentLength != req.ContentLength {
		slog.Error("mismatched content length", "contentLength", contentLength)
		return nil, fmt.Errorf("mismatched content length")
	}

	return &req, nil
}

func pathAndQueryParamsFromRequestTarget(requestTarget string) (string, map[string]string) {
	splitRequestTarget := strings.SplitN(requestTarget, "?", 2)
	path := splitRequestTarget[0]

	queryParams := make(map[string]string)

	if len(splitRequestTarget) != 2 {
		return path, queryParams
	}

	queryParamsRaw := strings.Split(splitRequestTarget[1], "&")
	for _, queryParam := range queryParamsRaw {
		splitQueryParam := strings.SplitN(queryParam, "=", 2)
		queryParams[splitQueryParam[0]] = splitQueryParam[1]
	}

	return path, queryParams
}
