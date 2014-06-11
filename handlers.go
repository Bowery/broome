// Copyright 2013-2014 Bowery, Inc.
// Contains http handlers that implement 404's, loggers, and other useful handlers.
package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// NotFoundHandler just responds with a 404 and a message.
var NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	res.Body["error"] = http.StatusText(http.StatusNotFound)
	res.Send(http.StatusNotFound)
})

// SlashHandler is a http.Handler that removes trailing slashes.
type SlashHandler struct {
	Handler http.Handler
}

// ServeHTTP strips trailing slashes and calls the handler.
func (sh *SlashHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		req.URL.Path = strings.TrimRight(req.URL.Path, "/")
		req.RequestURI = req.URL.RequestURI()
	}

	sh.Handler.ServeHTTP(rw, req)
}

// LogHandler is a http.Handler that logs requests in a simple format.
type LogHandler struct {
	Writer  io.Writer
	Handler http.Handler
}

// ServeHTTP logs the request and calls the handler.
func (lh *LogHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	accessTime := time.Now()
	loggedWriter := &responseLogger{ResponseWriter: rw}

	lh.Handler.ServeHTTP(loggedWriter, req)

	content := req.Method + " " + req.URL.String() + " " +
		strconv.Itoa(loggedWriter.Status) + " " + time.Since(accessTime).String()
	lh.Writer.Write([]byte(content + "\n"))
}

// responseLogger is a http.ResponseWriter, it keeps the state of the responses
// status code.
type responseLogger struct {
	ResponseWriter http.ResponseWriter
	Status         int
}

// Header returns the responses headers.
func (rl *responseLogger) Header() http.Header {
	return rl.ResponseWriter.Header()
}

// WriteHeader writes the head, and keeps track of status code.
func (rl *responseLogger) WriteHeader(status int) {
	rl.ResponseWriter.WriteHeader(status)
	rl.Status = status
}

// Write writes the response.
func (rl *responseLogger) Write(b []byte) (int, error) {
	// If no status has been written, default to OK.
	if rl.Status == 0 {
		rl.Status = http.StatusOK
	}

	return rl.ResponseWriter.Write(b)
}
