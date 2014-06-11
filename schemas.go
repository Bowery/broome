// Copyright 2013-2014 Bowery, Inc.
package main

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/orchestrate-io/gorc"
	"time"
)

var orchestrate *gorc.Client

var UserCollection = "users"

func init() {
	orchestrate = gorc.NewClient("d10728b1-fb0d-4f02-b778-c0d6f3c725ff")
}

type User struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name,omitempty"`
	Email               string    `json:"email,omitempty"`
	Password            string    `json:"password,omitempty"`
	Salt                string    `json:"salt,omitempty"`
	Token               string    `json:"token,omitempty"`
	IsAdmin             bool      `json:"isAdmin,omitempty"`
	StripeToken         string    `json:"stripeToken,omitempty"`
	NextPaymentTime     time.Time `json:"nextPaymentTime,omitempty"`
	IntegrationEngineer string    `json:"integrationEngineer,omitempty"`
	CreatedAt           time.Time `json:"createdAt,omitempty"`
	LastActiveAt        time.Time `json:"lastActiveAt,omitempty"`
}
