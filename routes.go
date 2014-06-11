// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for crosby server.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Bowery/broadway/db"
	"github.com/bradrydzewski/go.stripe"
	"github.com/gorilla/mux"
)

// 32 MB, same as http.
const httpMaxMem = 32 << 10

var STATIC_DIR string = TEMPLATE_DIR

// Route is a single named route with a http.HandlerFunc.
type Route struct {
	Path    string
	Methods []string
	Handler http.HandlerFunc
}

// List of named routes.
var Routes = []*Route{
	&Route{"/", []string{"GET"}, HomeHandler},
	&Route{"/signup", []string{"GET"}, SignUpHandler},
	&Route{"/thanks!", []string{"GET"}, ThanksHandler},
	&Route{"/healthz", []string{"GET"}, HealthzHandler},
	&Route{"/static/{rest}", []string{"GET"}, http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))).ServeHTTP},
}

func init() {
	stripeKey := "sk_test_BKnPoMNUWSGHJsLDcSGeV8I9"
	var cwd, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	if os.Getenv("ENV") == "production" {
		STATIC_DIR = cwd + "/" + STATIC_DIR
		stripeKey = "sk_live_fx0WR9yUxv6JLyOcawBdNEgj"
	}
	stripe.SetKey(stripeKey)
}

// GET /, Introduction to Crosby
func HomeHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "home", map[string]string{"Name": "Crosby"}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// POST /session, Creates a new user and charges them for the first year.
func CreateSessionHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	if err := req.ParseForm(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	name := req.PostFormValue("name")
	email := req.PostFormValue("stripeEmail")
	if email == "" {
		email = req.PostFormValue("email")
	}

	u := &db.Developer{
		Name:            name,
		Email:           email,
		NextPaymentTime: time.Now().Add(time.Hour * 24 * 30),
	}

	// Silent Signup from cli and not signup form. Will not charge them, but will give them a free month
	if req.PostFormValue("stripeToken") == "" || req.PostFormValue("stripeEmail") == "" || req.PostFormValue("password") == "" {
		if err := u.Save(); err != nil {
			res.Body["status"] = "failed"
			res.Body["err"] = err.Error()
			res.Send(http.StatusBadRequest)
			return
		}
		res.Body["status"] = "created"
		res.Body["user"] = u
		res.Send(http.StatusOK)
		return
	}

	// Use Account Number (Id) to get user
	id := req.PostFormValue("id")
	if id == "" {
		res.Body["status"] = "failed"
		res.Body["err"] = "Missing required field: id"
		res.Send(http.StatusBadRequest)
		return
	}

	u, err := db.GetDeveloperById(id)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["err"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}
	u.Name = name
	u.Email = email
	u.NextPaymentTime = time.Now().Add(time.Hour * 24 * 30)

	// Hash Password
	u.Salt, err = HashToken()
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}
	u.Password = HashPassword(req.PostFormValue("password"), u.Salt)

	// Create Stripe Customer
	customerParams := stripe.CustomerParams{
		Email: u.Email,
		Desc:  u.Name,
		Token: req.PostFormValue("stripeToken"),
	}
	customer, err := stripe.Customers.Create(&customerParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	// Charge Stripe Customer
	chargeParams := stripe.ChargeParams{
		Desc:     "Crosby Annual License",
		Amount:   2500,
		Currency: "usd",
		Customer: customer.Id,
	}
	_, err = stripe.Charges.Create(&chargeParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	// Update Stripe Info and Persist to Orchestrate
	u.StripeToken = customer.Id
	if err := u.Save(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	if req.PostFormValue("html") != "" {
		http.Redirect(rw, req, "/thanks!", 302)
		return
	}

	res.Body["status"] = "success"
	res.Body["user"] = u
	res.Send(http.StatusOK)
}

// GET /session/{id}, Gets user by ID. If their license has expired it attempts
// to charge them again. It is called everytime crosby is run.
func SessionHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)

	id := mux.Vars(req)["id"]
	fmt.Println("Getting user by id", id)
	u, err := db.GetDeveloperById(id)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	if u.NextPaymentTime.After(time.Now()) {
		res.Body["status"] = "found"
		res.Body["user"] = u
		res.Send(http.StatusOK)
		return
	}

	if u.StripeToken == "" {
		res.Body["status"] = "expired"
		res.Body["user"] = u
		res.Send(http.StatusOK)
		return
	}

	// Charge them, update expiration, & respond with found.
	// Charge Stripe Customer
	chargeParams := stripe.ChargeParams{
		Desc:     "Crosby Annual License",
		Amount:   2500,
		Currency: "usd",
		Customer: u.StripeToken,
	}
	_, err = stripe.Charges.Create(&chargeParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}
	u.NextPaymentTime = time.Now()
	if err := u.Save(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	res.Body["status"] = "found"
	res.Body["user"] = u
	res.Send(http.StatusOK)
	return
}

// GET /signup, Renders signup find. Will also handle billing
func SignUpHandler(w http.ResponseWriter, req *http.Request) {
	stripePubKey := "pk_test_m8TQEAkYWSc1jZh7czo8xhA7"
	if os.Getenv("ENV") == "production" {
		stripePubKey = "pk_live_LOngSSK6d3qwW0aBEhWSVEcF"
	}

	if err := RenderTemplate(w, "signup", map[string]interface{}{
		"isSignup":     true,
		"stripePubKey": stripePubKey,
	}); err != nil {
		RenderTemplate(w, "error", map[string]string{"Error": err.Error()})
	}
}

// Get /thanks!, Renders a thank you/confirmation message stored in static/thanks.html
func ThanksHandler(w http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(w, "thanks", map[string]interface{}{}); err != nil {
		RenderTemplate(w, "error", map[string]string{"Error": err.Error()})
	}
}
func HealthzHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "ok")
}
