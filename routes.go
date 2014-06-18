// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for broome server.
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Bowery/broome/db"
	"github.com/Bowery/broome/util"
	"github.com/bradrydzewski/go.stripe"
	"github.com/gorilla/mux"
)

// 32 MB, same as http.
const (
	httpMaxMem = 32 << 10
	slackToken = "xoxp-2157690968-2174706611-2385261803-c58929"
)

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
	&Route{"/developers", []string{"GET"}, AdminHandler},
	&Route{"/developers", []string{"POST"}, CreateDeveloperHandler},
	&Route{"/developers/{token}", []string{"PUT"}, DeveloperEditHandler},
	&Route{"/developers/{token}", []string{"GET"}, DeveloperInfoHandler},
	&Route{"/developers/new", []string{"GET"}, NewDevHandler},
	&Route{"/signup/{id}", []string{"GET"}, SignUpHandler},
	&Route{"/thanks!", []string{"GET"}, ThanksHandler},
	&Route{"/healthz", []string{"GET"}, HealthzHandler},
	&Route{"/static/{rest}", []string{"GET"}, StaticHandler},
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

// GET /developers, Admin Interface that lists developers
func AdminHandler(rw http.ResponseWriter, req *http.Request) {
	ds, err := db.GetDevelopers(map[string]interface{}{})
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	if err := RenderTemplate(rw, "admin", map[string][]*db.Developer{
		"Developers": ds,
	}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /developers/{token}, Admin Interface for a single developer
func DeveloperInfoHandler(rw http.ResponseWriter, req *http.Request) {
	token := mux.Vars(req)["token"]
	fmt.Println(token)

	d, err := db.GetDeveloper(map[string]interface{}{"token": token})
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}
	RenderTemplate(rw, "developer", d)
}

// PUT /developers/{token}, edits a developer
func DeveloperEditHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	params := mux.Vars(req)
	token := params["token"]
	if token == "" {
		res.Body["status"] = "failed"
		res.Body["error"] = "missing token"
		res.Send(http.StatusBadRequest)
		return
	}

	if err := req.ParseForm(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	query := map[string]interface{}{}
	query["token"] = token
	update := map[string]interface{}{}

	dev, err := db.GetDeveloper(query)
	if err != nil {
		res.Body["status"] = "failed"
		res.Send(http.StatusInternalServerError)
		return
	}

	if password := req.FormValue("password"); password != "" {
		update["password"] = util.HashPassword(password, dev.Salt)
	}

	if isAdmin := req.FormValue("isAdmin"); isAdmin == "true" {
		update["isAdmin"] = true
	} else {
		update["isAdmin"] = false
	}

	// TODO add datetime parsing
	for _, field := range []string{"name", "email", "nextPaymentTime", "integrationEngineer"} {
		val := req.FormValue(field)
		if val != "" {
			update[field] = val
		}
	}

	fmt.Print("update -> ")
	fmt.Println(update)

	if err := db.UpdateDeveloper(query, update); err != nil {
		res.Body["status"] = "failed"
		res.Send(http.StatusInternalServerError)
		return
	}

	res.Body["status"] = "updated"
	res.Body["update"] = update
	res.Send(http.StatusOK)
}

// POST /developers, Creates a new developer
func CreateDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	res := NewResponder(rw, req)

	dev := &db.Developer{
		Name:  params["name"],
		Email: params["email"],
	} // password?

	// Post to slack
	if os.Getenv("ENV") == "production" {
		payload := url.Values{}
		payload.Set("token", slackToken)
		payload.Set("channel", "#users")
		payload.Set("text", dev.Name+" "+dev.Email+" just signed up.")
		payload.Set("username", "Drizzy Drake")
		http.PostForm("https://slack.com/api/chat.postMessage", payload)
	}

	res.Body["status"] = "todo"
	res.Body["developer"] = dev
	res.Send(http.StatusOK)

}

// GET /developers/new, Admin helper for creating developers
func NewDevHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "new", map[string]string{}); err != nil {
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
	u.Salt = util.HashToken()
	u.Password = util.HashPassword(req.PostFormValue("password"), u.Salt)

	// // Create Stripe Customer
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

// GET /signup/:id, Renders signup find. Will also handle billing
func SignUpHandler(w http.ResponseWriter, req *http.Request) {
	stripePubKey := "pk_test_m8TQEAkYWSc1jZh7czo8xhA7"
	if os.Getenv("ENV") == "production" {
		stripePubKey = "pk_live_LOngSSK6d3qwW0aBEhWSVEcF"
	}

	if err := RenderTemplate(w, "signup", map[string]interface{}{
		"isSignup":     true,
		"stripePubKey": stripePubKey,
		"id":           mux.Vars(req)["id"],
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

// GET /healthz, Indicates that the service is up
func HealthzHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "ok")
}

func StaticHandler(res http.ResponseWriter, req *http.Request) {
	http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))).ServeHTTP(res, req)
}
