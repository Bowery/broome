// Copyright 2013-2014 Bowery, Inc.
// Contains routines to send JSON responses to requests.
package main

import (
	"encoding/json"
	"net/http"
)

// Responder represents a structure that marshals JSON for a response.
type Responder struct {
	RW   http.ResponseWriter
	Req  *http.Request
	Body map[string]interface{}
}

// NewResponder creates a new responder with an empty body.
func NewResponder(rw http.ResponseWriter, req *http.Request) *Responder {
	return &Responder{rw, req, map[string]interface{}{}}
}

// Send the response marshalling data to JSON.
func (res *Responder) Send(status int) {
	if status != http.StatusOK {
		res.Body["status"] = "failed"
	}
	res.RW.Header().Set("Content-Type", "application/json")

	// Marshal JSON, if failed send a raw json string.
	contents, err := json.Marshal(res.Body)
	if err != nil {
		status = http.StatusInternalServerError
		contents = []byte("{\"status\":\"failed\",\"error\":\"" + err.Error() + "\"}")
	}

	res.RW.WriteHeader(status)
	res.RW.Write(contents)
}
