// Copyright 2013-2014 Bowery, Inc.
// Contains the main entry point
package main

import (
	"os"

	"github.com/Bowery/gopackages/config"
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
		&web.StatHandler{Key: config.StatHatKey, Name: "broome"},
	}, Routes)
	server.Router.NotFoundHandler = &web.NotFoundHandler{renderer}
	server.AuthHandler = &web.AuthHandler{Auth: AuthHandler}
	server.ListenAndServe()
}
