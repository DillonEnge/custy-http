# custy-http

A custom, lightweight HTTP server library written in Go with a focus on simplicity and control.

## Overview

`custy-http` is a barebones HTTP server implementation built from scratch using only Go's standard library. It provides direct control over HTTP request and response handling without the overhead of larger frameworks.

## Features

- Pure Go implementation using only standard library
- Support for common HTTP methods (GET, POST, PUT, PATCH, DELETE, OPTIONS)
- Simple router interface for registering handlers
- Configurable read and write timeouts
- Concurrent connection handling
- Low-level control over HTTP messages

## Installation

```bash
go get github.com/DillonEnge/custy-http
```

## Usage

Here's a simple example of how to use `custy-http`:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DillonEnge/custy-http"
)

func main() {
	// Create a new server with options
	server := custyhttp.NewServer(
		custyhttp.WithReadTimeout(5 * time.Second),
		custyhttp.WithWriteTimeout(5 * time.Second),
	)

	// Create a router
	router := custyhttp.NewRouter(server)

	// Register routes
	router.Get("/hello", helloHandler)

	// Start the server
	ctx := context.Background()
	if err := server.ListenAndServe(ctx, ":8080"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func helloHandler(ctx context.Context, req *custyhttp.Request) (*custyhttp.Response, error) {
	// Create response
	resp := &custyhttp.Response{
		StatusCode:  200,
		StatusText:  "OK",
		Protocol:    "HTTP/1.1",
		ContentType: "text/plain",
		Body:        []byte("Hello, World!"),
	}
	
	return resp, nil
}
```

## Motivation

`custy-http` was developed to provide a deeper understanding of HTTP server implementation while maintaining a simple API. It's designed for developers who want more control over the HTTP layer without the abstraction of larger frameworks.

## License

MIT