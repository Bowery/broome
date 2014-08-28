// Copyright 2013-2014 Bowery, Inc.
// Contains http handlers that implement 404's, loggers, and other useful handlers.
package main

import (
	"encoding/base64"
	"fmt"
	"github.com/Bowery/broome/db"
	"github.com/Bowery/broome/util"
	"io"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// NotFoundHandler just responds with a 404 and a message.
var NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	res := &Responder{rw, req, map[string]interface{}{}}
	res.Body["error"] = http.StatusText(http.StatusNotFound)
	res.Send(http.StatusNotFound)
})

// AuthHandler is a http.Handler that checks token is valid
type AuthHandler struct {
	Handler http.Handler
}

func ForceLogin(rw http.ResponseWriter) {
	rw.Header().Set("WWW-Authenticate", "Basic realm=\"localhost\"")
	http.Error(rw, http.StatusText(401), 401)
}

func (ah *AuthHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Skip Auth for Dev Creation
	if req.URL.Path == "/developers" && req.Method == "POST" {
		fmt.Println("Skipping Auth Check. Creating New Developer")
		ah.Handler.ServeHTTP(rw, req)
		return
	}

	h, ok := req.Header["Authorization"]
	if !ok || len(h) == 0 {
		fmt.Println("here1")
		ForceLogin(rw)
		return
	}

	parts := strings.SplitN(h[0], " ", 2)
	scheme := parts[0]
	if scheme != "Basic" {
		fmt.Println("Auth type is not supported")
		ForceLogin(rw)
		return
	}

	b, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Println(err)
		ForceLogin(rw)
		return
	}

	credentials := strings.Split(string(b), ":")
	if len(credentials) != 2 {
		fmt.Println("Auth Failed: Credential Format Invalid")
		ForceLogin(rw)
		return
	}

	username := credentials[0]
	password := credentials[1]
	query := bson.M{}
	if password == "" {
		query["token"] = username
	} else {
		query["email"] = username
	}

	dev, err := db.GetDeveloper(query)
	if err != nil || dev.ID == "" {
		fmt.Println("Auth Failed: User not found.")
		ForceLogin(rw)
		return
	}

	if password != "" && dev.Password != util.HashPassword(password, dev.Salt) {
		fmt.Println("Auth Failed: Invalid Password.")
		ForceLogin(rw)
		return
	}

	ah.Handler.ServeHTTP(rw, req)
}

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

// CorsHandler is a http.Handler that enabled cross origin resource sharing.
type CorsHandler struct {
	Handler http.Handler
}

func (ch *CorsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Access-Control-Allow-Origin", "*")
	ch.Handler.ServeHTTP(rw, req)
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
