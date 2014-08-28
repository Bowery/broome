// Copyright 2013-2014 Bowery, Inc.
// Contains the main entry point
package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

func Handler() http.Handler {
	router := mux.NewRouter()
	router.NotFoundHandler = NotFoundHandler

	for _, r := range Routes {
		route := router.NewRoute()
		route.Path(r.Path).Methods(r.Methods...)
		if r.Auth {
			h := &AuthHandler{r.Handler}
			r.Handler = h.ServeHTTP
		}
		route.HandlerFunc(r.Handler)
	}
	return &SlashHandler{&CorsHandler{&LogHandler{os.Stdout, router}}}
}

func main() {

	port := ":4000"
	if os.Getenv("ENV") == "production" {
		port = ":80"
	}

	// Start the server.
	server := &http.Server{
		Addr:    port,
		Handler: Handler(),
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
