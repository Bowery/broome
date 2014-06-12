// Copyright 2013-2014 Bowery, Inc.
// Contains the main entry point
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.NotFoundHandler = NotFoundHandler
	for _, r := range Routes {
		route := router.NewRoute()
		route.Path(r.Path).Methods(r.Methods...)
		route.HandlerFunc(r.Handler)
	}

	port := ":4000"
	if os.Getenv("ENV") == "production" {
		port = ":80"
	}

	// Start the server.
	server := &http.Server{
		Addr:    port,
		Handler: &SlashHandler{&LogHandler{os.Stdout, router}},
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
