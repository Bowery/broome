// Copyright 2013-2014 Bowery, Inc.
// Contains the main entry point
package main

import (
	"os"

	"github.com/Bowery/gopackages/web"
)

func main() {
	port := ":4000"
	if os.Getenv("ENV") == "production" {
		port = ":80"
	}

	server := web.NewServer(port, []web.Handler{
		new(web.SlashHandler),
		new(web.CorsHandler),
	}, Routes)
	server.Router.NotFoundHandler = &web.NotFoundHandler{r}
	server.AuthHandler = &web.AuthHandler{Auth: AuthHandler}
	server.ListenAndServe()
}
