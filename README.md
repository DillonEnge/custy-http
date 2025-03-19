# custy-http

A custom, lightweight HTTP server and client library written in Go with a focus on simplicity and control.

## Overview

`custy-http` is a barebones HTTP/1.x implementation built from scratch using only Go's standard library. It provides direct control over HTTP request and response handling without the overhead of larger frameworks.

## Features

- Pure Go implementation using only standard library
- Support for common HTTP methods (GET, POST, PUT, PATCH, DELETE, OPTIONS)
- Simple router interface for registering handlers
- Configurable read and write timeouts
- Concurrent connection handling
- Low-level control over HTTP messages
- HTTP client for making requests

## Installation

```bash
go get github.com/DillonEnge/custy-http
```

## Usage

### Server Example

Here's an example of setting up an HTTP server with `custy-http`:

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

	// Register routes for different HTTP methods
	router.Get("/hello", helloHandler)
	router.Post("/users", createUserHandler)
	router.Put("/users", updateUserHandler)
	router.Delete("/users", deleteUserHandler)

	// Start the server
	ctx := context.Background()
	fmt.Println("Server starting on :8080...")
	if err := server.ListenAndServe(ctx, ":8080"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func helloHandler(ctx context.Context, req *custyhttp.Request) (*custyhttp.Response, error) {
	// Access query parameters
	params := req.QueryParams()
	name := "World"
	if val, ok := params["name"]; ok {
		name = val
	}

	// Create response
	resp := &custyhttp.Response{
		StatusCode:  200,
		ContentType: "text/plain",
		Body:        []byte(fmt.Sprintf("Hello, %s!", name)),
	}
	
	return resp, nil
}

func createUserHandler(ctx context.Context, req *custyhttp.Request) (*custyhttp.Response, error) {
	// Access request body
	reqBody := req.Body()
	
	// Process request...
	
	resp := &custyhttp.Response{
		StatusCode:  201,
		ContentType: "application/json",
		Body:        []byte(`{"status":"created"}`),
	}
	
	return resp, nil
}

func updateUserHandler(ctx context.Context, req *custyhttp.Request) (*custyhttp.Response, error) {
	// Implementation...
	resp := &custyhttp.Response{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"status":"updated"}`),
	}
	
	return resp, nil
}

func deleteUserHandler(ctx context.Context, req *custyhttp.Request) (*custyhttp.Response, error) {
	// Implementation...
	resp := &custyhttp.Response{
		StatusCode:  204,
	}
	
	return resp, nil
}
```

### Client Example

Here's an example of using the `custy-http` client to make HTTP requests:

```go
package main

import (
	"fmt"

	"github.com/DillonEnge/custy-http"
)

func main() {
	// Create a new HTTP client
	client := custyhttp.NewClient("localhost:8080")
	
	// Prepare a GET request
	getRequest := &custyhttp.Request{
		RequestTarget: "/hello?name=John",
		ContentType:   "application/json",
	}
	
	// Send the GET request
	getResponse, err := client.Do(custyhttp.METHOD_GET, getRequest)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	
	fmt.Printf("GET Response: %s\n", getResponse.Body())
	
	// Prepare a POST request with a body
	postRequest := &custyhttp.Request{
		RequestTarget: "/users",
		ContentType:   "application/json",
	}
	postRequest.WriteBytesToBody([]byte(`{"name":"John","email":"john@example.com"}`))
	
	// Send the POST request
	postResponse, err := client.Do(custyhttp.METHOD_POST, postRequest)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	
	fmt.Printf("POST Response: %s\n", postResponse.Body())
}
```

## Motivation

`custy-http` was developed to provide a deeper understanding of HTTP server implementation while maintaining a simple API. It's designed for developers who want more control over the HTTP layer without the abstraction of larger frameworks.

## License

MIT