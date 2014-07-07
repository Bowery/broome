// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for broome server.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bowery/broome/db"
	"github.com/Bowery/broome/requests"
	"github.com/Bowery/broome/util"
	"github.com/bradrydzewski/go.stripe"
	"github.com/gorilla/mux"
	"github.com/mattbaird/gochimp"
)

// 32 MB, same as http.
const (
	httpMaxMem = 32 << 10
	slackToken = "xoxp-2157690968-2174706611-2385261803-c58929"
)

var (
	STATIC_DIR string = TEMPLATE_DIR
	chimp      *gochimp.ChimpAPI
	mandrill   *gochimp.MandrillAPI
)

// Route is a single named route with a http.HandlerFunc.
type Route struct {
	Path    string
	Methods []string
	Handler http.HandlerFunc
	Auth    bool
}

// List of named routes.
var Routes = []*Route{
	&Route{"/", []string{"GET"}, HomeHandler, true},
	&Route{"/developers", []string{"GET"}, AdminHandler, true},
	&Route{"/developers", []string{"POST"}, CreateDeveloperHandler, false},
	&Route{"/developers/token", []string{"POST"}, CreateTokenHandler, false},
	&Route{"/developers/me", []string{"GET"}, GetCurrentDeveloperHandler, false},
	&Route{"/developers/{token}", []string{"PUT"}, UpdateDeveloperHandler, true},
	&Route{"/developers/{token}", []string{"GET"}, DeveloperInfoHandler, true},
	&Route{"/developers/new", []string{"GET"}, NewDevHandler, true},
	&Route{"/signup/{id}", []string{"GET"}, SignUpHandler, false},
	&Route{"/thanks!", []string{"GET"}, ThanksHandler, false},
	&Route{"/reset/{email}", []string{"GET"}, ResetPasswordHandler, false},
	&Route{"/developers/reset/{token}/{id}", []string{"GET"}, ResetHandler, false},
	&Route{"/developers/reset/{token}", []string{"PUT"}, PasswordEditHandler, false},
	&Route{"/healthz", []string{"GET"}, HealthzHandler, false},
	&Route{"/static/{rest}", []string{"GET"}, StaticHandler, false},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stripeKey := "sk_test_BKnPoMNUWSGHJsLDcSGeV8I9"
	chimpKey := "923742397a5bf0c8e3efc6d78517911d-us3"
	mandrillKey := "nYs-WjIVVEAo4ELuda8Elw" // "deMcwBJQFPC7FLeDZwlErg" // "DfJcUPXNJDTYQOYN0jNcGg"
	var cwd, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	if os.Getenv("ENV") == "production" {
		STATIC_DIR = cwd + "/" + STATIC_DIR
		stripeKey = "sk_live_fx0WR9yUxv6JLyOcawBdNEgj"
	}
	stripe.SetKey(stripeKey)
	chimp = gochimp.NewChimp(chimpKey, true)
	mandrill, _ = gochimp.NewMandrill(mandrillKey)
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
	d, err := db.GetDeveloper(map[string]interface{}{"token": token})
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	marshalledTime, _ := d.NextPaymentTime.MarshalJSON()

	RenderTemplate(rw, "developer", map[string]interface{}{
		"Token":               d.Token,
		"Name":                d.Name,
		"Email":               d.Email,
		"IsAdmin":             d.IsAdmin,
		"NextPaymentTime":     string(marshalledTime[1 : len(marshalledTime)-1]), // trim inexplainable quotes and Z at the end that breaks shit
		"IntegrationEngineer": d.IntegrationEngineer,
	})
}

// PUT /developers/{token}, edits a developer
func UpdateDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	token := mux.Vars(req)["token"]
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

	query := map[string]interface{}{"token": token}
	update := map[string]interface{}{}

	u, err := db.GetDeveloper(query)
	if err != nil {
		res.Body["status"] = "failed"
		res.Send(http.StatusInternalServerError)
		return
	}

	if password := req.FormValue("password"); password != "" {
		update["password"] = util.HashPassword(password, u.Salt)
	}

	if nextPaymentTime := req.FormValue("nextPaymentTime"); nextPaymentTime != "" {
		update["nextPaymentTime"], err = time.Parse(time.RFC3339, nextPaymentTime)
	}

	if isAdmin := req.FormValue("isAdmin"); isAdmin != "" {
		update["isAdmin"] = isAdmin == "on" || isAdmin == "true"
	}

	// TODO add datetime parsing
	for _, field := range []string{"name", "email", "integrationEngineer"} {
		val := req.FormValue(field)
		if val != "" {
			update[field] = val
		}
	}

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
	res := NewResponder(rw, req)

	type engineer struct {
		Name  string
		Email string
	}

	integrationEngineers := []*engineer{
		&engineer{Name: "Steve Kaliski", Email: "steve@bowery.io"},
		&engineer{Name: "David Byrd", Email: "byrd@bowery.io"},
		&engineer{Name: "Larz Conwell", Email: "larz@bowery.io"},
		&engineer{Name: "Ricky Medina", Email: "rm@bowery.io"},
	}

	integrationEngineer := integrationEngineers[rand.Int()%len(integrationEngineers)]

	var body requests.LoginReq

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	if body.Email == "" || body.Password == "" {
		res.Body["status"] = "failed"
		res.Body["error"] = "Email and Password Required."
		res.Send(http.StatusBadRequest)
		return
	}

	u := &db.Developer{
		Name:                body.Name,
		Email:               body.Email,
		Password:            body.Password,
		IntegrationEngineer: integrationEngineer.Name,
		IsPaid:              false,
	}

	if os.Getenv("ENV") == "production" { // already checked that email wasn't empty
		if _, err := chimp.ListsSubscribe(gochimp.ListsSubscribe{
			ListId: "200e892f56",
			Email:  gochimp.Email{Email: u.Email},
		}); err != nil {
			res.Body["status"] = "failed"
			res.Body["error"] = err.Error()
			res.Send(http.StatusBadRequest)
			return
		}

		message, err := RenderEmail("welcome", map[string]interface{}{
			"name":     strings.Split(u.Name, " ")[0],
			"engineer": integrationEngineer,
		})

		if err != nil {
			res.Body["status"] = "failed"
			res.Body["error"] = err.Error()
			res.Send(http.StatusBadRequest)
			return
		}

		_, err = mandrill.MessageSend(gochimp.Message{
			Subject:   "Welcome and Meet Your Integration Engineer",
			FromEmail: integrationEngineer.Email,
			FromName:  integrationEngineer.Name,
			To: []gochimp.Recipient{{
				Email: u.Email,
				Name:  u.Name,
			}},
			Html: message,
		}, false)

		if err != nil {
			res.Body["status"] = "failed"
			res.Body["error"] = err.Error()
			res.Send(http.StatusBadRequest)
			return
		}
	}

	if err := u.Save(); err != nil {
		res.Body["status"] = "failed"
		res.Body["err"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	// Post to slack
	if os.Getenv("ENV") == "production" {
		payload := url.Values{}
		payload.Set("token", slackToken)
		payload.Set("channel", "#users")
		payload.Set("text", u.Name+" "+u.Email+" just signed up.")
		payload.Set("username", "Drizzy Drake")
		http.PostForm("https://slack.com/api/chat.postMessage", payload)
	}

	res.Body["status"] = "created"
	res.Body["developer"] = u

	res.Send(http.StatusOK)
}

// GET /developers/new, Admin helper for creating developers
func NewDevHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "new", map[string]string{}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// POST /developer/token, logs in a user by creating a new token
func CreateTokenHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	var body requests.LoginReq
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	email := body.Email
	password := body.Password
	if email == "" || password == "" {
		res.Body["status"] = "failed"
		res.Body["error"] = "Email and Password Required."
		res.Send(http.StatusBadRequest)
		return
	}

	query := map[string]interface{}{"email": email}
	u, err := db.GetDeveloper(query)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = "No such developer with email " + email + "."
		res.Send(http.StatusInternalServerError)
		return
	}

	if util.HashPassword(password, u.Salt) != u.Password {
		res.Body["status"] = "failed"
		res.Body["error"] = "Incorrect Password"
		res.Send(http.StatusInternalServerError)
		return
	}

	token := util.HashToken()

	update := map[string]interface{}{"token": token}
	if err := db.UpdateDeveloper(query, update); err != nil {
		res.Body["status"] = "failed"
		res.Send(http.StatusInternalServerError)
		return
	}
	res.Body["status"] = "created"
	res.Body["token"] = token
	res.Send(http.StatusOK)
}

// GET /developers/me, return the logged in developer
func GetCurrentDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)

	if err := req.ParseForm(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	token := req.FormValue("token")
	if token == "" {
		res.Body["status"] = "failed"
		res.Body["error"] = "Valid token required."
		res.Send(http.StatusInternalServerError)
		return
	}

	query := map[string]interface{}{"token": token}
	u, err := db.GetDeveloper(query)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusInternalServerError)
		return
	}

	res.Body["status"] = "found"
	res.Body["developer"] = u
	res.Send(http.StatusOK)
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

// GET /signup/:id, Renders signup find. Will also handle billing
func SignUpHandler(rw http.ResponseWriter, req *http.Request) {
	stripePubKey := "pk_test_m8TQEAkYWSc1jZh7czo8xhA7"
	if os.Getenv("ENV") == "production" {
		stripePubKey = "pk_live_LOngSSK6d3qwW0aBEhWSVEcF"
	}

	if err := RenderTemplate(rw, "signup", map[string]interface{}{
		"isSignup":     true,
		"stripePubKey": stripePubKey,
		"id":           mux.Vars(req)["id"],
	}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /thanks!, Renders a thank you/confirmation message stored in static/thanks.html
func ThanksHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "thanks", map[string]interface{}{}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /reset/{email}, Request link to reset password--emails user
func ResetPasswordHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)

	// TODO check empty token
	email := mux.Vars(req)["email"]
	if email == "" {
		res.Body["status"] = "failed"
		res.Body["error"] = "no email provided"
		res.Send(http.StatusBadRequest)
		return
	}

	u, err := db.GetDeveloper(map[string]interface{}{"email": email})
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	message, err := RenderEmail("password_email", map[string]interface{}{
		"name":     strings.Split(u.Name, " ")[0],
		"id":       u.ID.Hex(),
		"token":    u.Token,
		"engineer": u.IntegrationEngineer,
	})
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	_, err = mandrill.MessageSend(gochimp.Message{
		Subject:   "Bowery Password Reset",
		FromEmail: "support@bowery.io",
		FromName:  "Bowery Support",
		To: []gochimp.Recipient{{
			Email: u.Email,
			Name:  u.Name,
		}},
		Html: message,
	}, false)

	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	res.Body["status"] = "success"
	res.Send(http.StatusOK)
}

// GET /developers/{token}/reset/{id}, Serves from where users can reset their password.
func ResetHandler(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	token := mux.Vars(req)["token"]

	u, err := db.GetDeveloperById(id)
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	if token != u.Token {
		RenderTemplate(rw, "error", map[string]string{"Error": "Invalid Token"})
		return
	}

	if err := RenderTemplate(rw, "password_reset", map[string]interface{}{
		"Token": u.Token,
		"ID":    u.ID.Hex(),
	}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// PUT /developers/{token}/reset, Edit password
func PasswordEditHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)

	if err := req.ParseForm(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	id := req.FormValue("id")
	u, err := db.GetDeveloperById(id)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["err"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	if util.HashPassword(req.FormValue("old"), u.Salt) != u.Password {
		res.Body["status"] = "failed"
		res.Body["err"] = "Old password is incorrect."
		res.Send(http.StatusBadRequest)
		return
	}

	query := map[string]interface{}{"token": mux.Vars(req)["token"]}
	update := map[string]interface{}{"password": util.HashPassword(req.FormValue("new"), u.Salt)}
	if err := db.UpdateDeveloper(query, update); err != nil {
		res.Body["status"] = "failed"
		res.Body["err"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	res.Body["status"] = "success"
	res.Body["user"], err = json.Marshal(u)
	if err != nil {
		res.Body["status"] = "failed"
		res.Send(http.StatusInternalServerError)
		return
	}
	res.Send(http.StatusOK)
}

// GET /healthz, Indicates that the service is up
func HealthzHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "ok")
}

func StaticHandler(res http.ResponseWriter, req *http.Request) {
	http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))).ServeHTTP(res, req)
}
