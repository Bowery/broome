// Copyright 2013-2014 Bowery, Inc.
package db

import (
	"os"

	"github.com/Bowery/gopackages/database"
)

var Client *database.Client

func init() {
	dbAddr := ""
	dbUsr := ""
	dbPass := ""

	if os.Getenv("ENV") == "development" || os.Getenv("ENV") == "testing" {
		dbAddr = "localhost:27017"
	}

	if os.Getenv("ENV") == "production" {
		dbAddr = "ec2-54-166-124-190.compute-1.amazonaws.com,ec2-54-211-48-143.compute-1.amazonaws.com,ec2-54-87-106-210.compute-1.amazonaws.com"
		dbUsr = "bowery"
		dbPass = "java$cript"
	}

	var err error
	Client, err = database.NewClient(dbAddr, "bowery", dbUsr, dbPass)
	if err != nil {
		panic(err)
	}
}
