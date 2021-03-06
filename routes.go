// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for broome server.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bowery/broome/db"
	"github.com/Bowery/gopackages/config"
	"github.com/Bowery/gopackages/requests"
	"github.com/Bowery/gopackages/schemas"
	"github.com/Bowery/gopackages/util"
	"github.com/Bowery/gopackages/web"
	"github.com/bradrydzewski/go.stripe"
	"github.com/gorilla/mux"
	"github.com/mattbaird/gochimp"
	"github.com/unrolled/render"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// 32 MB, same as http.
const (
	httpMaxMem = 32 << 10
)

var (
	STATIC_DIR      string = TEMPLATE_DIR
	chimp           *gochimp.ChimpAPI
	mandrill        *gochimp.MandrillAPI
	stripePublicKey string
)

var renderer = render.New(render.Options{
	IndentJSON:    true,
	IsDevelopment: true,
})

// List of named routes.
var Routes = []web.Route{
	{"GET", "/admin", HomeHandler, true},
	{"GET", "/admin/developers", AdminHandler, true},
	{"POST", "/developers", CreateDeveloperHandler, false},
	{"POST", "/developers/token", CreateTokenHandler, false},
	{"POST", "/developers/check-admin", CheckAdminHandler, false},
	{"GET", "/developers/me", GetCurrentDeveloperHandler, false},
	{"GET", "/developers/{id}", GetDeveloperByIDHandler, false},
	{"GET", "/admin/developers/new", NewDevHandler, true},
	{"PUT", "/developers/{token}", UpdateDeveloperHandler, true},
	{"GET", "/admin/developers/{token}", DeveloperInfoHandler, true},
	{"POST", "/developers/{token}/pay", PaymentHandler, false},
	{"GET", "/session/{id}", SessionInfoHandler, false},
	{"GET", "/admin/signup/{id}", SignUpHandler, false},
	{"POST", "/signup", CreateSessionHandler, false},
	{"GET", "/admin/thanks!", ThanksHandler, false},
	{"GET", "/reset/{email}", ResetPasswordHandler, false},
	{"GET", "/developers/reset/{token}/{id}", ResetHandler, false},
	{"PUT", "/developers/reset/{token}", PasswordEditHandler, false},
	{"GET", "/healthz", HealthzHandler, false},
	{"GET", "/static/{rest}", StaticHandler, false},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stripeSecretKey := config.StripeTestSecretKey
	stripePublicKey = config.StripeTestPublicKey

	var cwd, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	if os.Getenv("ENV") == "production" {
		STATIC_DIR = cwd + "/" + STATIC_DIR
		stripeSecretKey = config.StripeLiveSecretKey
		stripePublicKey = config.StripeLivePublicKey
	}
	stripe.SetKey(stripeSecretKey)
	chimp = gochimp.NewChimp(config.MailchimpKey, true)
	mandrill, _ = gochimp.NewMandrill(config.MandrillKey)
}

func AuthHandler(req *http.Request, user, pass string) (bool, error) {
	query := bson.M{}
	if pass == "" {
		query["token"] = user
	} else {
		query["email"] = user
	}

	dev, err := db.GetDeveloper(query)
	if err != nil || dev.ID == "" {
		return false, err
	}

	if pass != "" && dev.Password != util.HashPassword(pass, dev.Salt) {
		return false, nil
	}

	return true, nil
}

// GET /admin, Introduction
func HomeHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "home", map[string]string{"Name": "Broome"}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /admin/developers, Admin Interface that lists developers
func AdminHandler(rw http.ResponseWriter, req *http.Request) {
	ds, err := db.GetDevelopers(map[string]interface{}{})
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	if err := RenderTemplate(rw, "admin", map[string][]*schemas.Developer{
		"Developers": ds,
	}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /admin/developers/{token}, Admin Interface for a single developer
func DeveloperInfoHandler(rw http.ResponseWriter, req *http.Request) {
	token := mux.Vars(req)["token"]
	d, err := db.GetDeveloper(map[string]interface{}{"token": token})
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	marshalledTime, _ := d.Expiration.MarshalJSON()

	RenderTemplate(rw, "developer", map[string]interface{}{
		"Token":               d.Token,
		"Name":                d.Name,
		"Email":               d.Email,
		"IsAdmin":             d.IsAdmin,
		"IsPaid":              d.IsPaid,
		"NextPaymentTime":     string(marshalledTime[1 : len(marshalledTime)-1]), // trim inexplainable quotes and Z at the end that breaks shit
		"IntegrationEngineer": d.IntegrationEngineer,
	})
}

// PUT /developers/{token}, edits a developer
func UpdateDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	token := mux.Vars(req)["token"]
	if token == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "missing token",
		})
		return
	}

	if err := req.ParseForm(); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	query := map[string]interface{}{"token": token}
	update := map[string]interface{}{}

	u, err := db.GetDeveloper(query)
	if err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	if password := req.FormValue("password"); password != "" {
		oldpass := req.FormValue("oldpassword")
		if oldpass == "" || util.HashPassword(oldpass, u.Salt) != u.Password {
			renderer.JSON(rw, http.StatusBadRequest, map[string]string{
				"status": requests.StatusFailed,
				"error":  "Old password is incorrect.",
			})
			return
		}

		update["password"] = util.HashPassword(password, u.Salt)
	}

	if nextPaymentTime := req.FormValue("nextPaymentTime"); nextPaymentTime != "" {
		update["nextPaymentTime"], err = time.Parse(time.RFC3339, nextPaymentTime)
	}

	if isAdmin := req.FormValue("isAdmin"); isAdmin != "" {
		update["isAdmin"] = isAdmin == "on" || isAdmin == "true"
	}

	if isPaid := req.FormValue("isPaid"); isPaid != "" {
		update["isPaid"] = isPaid == "on" || isPaid == "true"
	}

	// TODO add datetime parsing
	for _, field := range []string{"name", "email", "integrationEngineer"} {
		val := req.FormValue(field)
		if val != "" {
			update[field] = val
		}
	}

	if err := db.UpdateDeveloper(query, update); err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status": requests.StatusUpdated,
		"update": update,
	})
}

// POST /developers, Creates a new developer
func CreateDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	type engineer struct {
		Name  string
		Email string
	}

	integrationEngineers := []*engineer{
		&engineer{Name: "Steve Kaliski", Email: "steve@bowery.io"},
		&engineer{Name: "David Byrd", Email: "byrd@bowery.io"},
		&engineer{Name: "Larz Conwell", Email: "larz@bowery.io"},
	}

	integrationEngineer := integrationEngineers[rand.Int()%len(integrationEngineers)]

	var body requests.LoginReq

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	if body.Email == "" || body.Password == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Email and Password Required.",
		})
		return
	}

	u := &schemas.Developer{
		Name:                body.Name,
		Email:               body.Email,
		Password:            body.Password,
		Token:               util.HashToken(),
		IntegrationEngineer: integrationEngineer.Name,
		IsPaid:              false,
		CreatedAt:           time.Now().UnixNano() / int64(time.Millisecond),
	}

	_, err = db.GetDeveloper(bson.M{"email": u.Email})
	if err == nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  "email already exists",
		})
		return
	}

	if os.Getenv("ENV") == "production" && !strings.Contains(body.Email, "@bowery.io") {
		if _, err := chimp.ListsSubscribe(gochimp.ListsSubscribe{
			ListId: "200e892f56",
			Email:  gochimp.Email{Email: u.Email},
		}); err != nil {
			renderer.JSON(rw, http.StatusBadRequest, map[string]string{
				"status": requests.StatusFailed,
				"error":  err.Error(),
			})
			return
		}

		message, err := RenderEmail("welcome", map[string]interface{}{
			"name":     strings.Split(u.Name, " ")[0],
			"engineer": integrationEngineer,
		})

		if err != nil {
			renderer.JSON(rw, http.StatusBadRequest, map[string]string{
				"status": requests.StatusFailed,
				"error":  err.Error(),
			})
			return
		}

		_, err = mandrill.MessageSend(gochimp.Message{
			Subject:   "Welcome to Bowery!",
			FromEmail: "hello@bowery.io",
			FromName:  integrationEngineer.Name,
			To: []gochimp.Recipient{{
				Email: u.Email,
				Name:  u.Name,
			}},
			Html: message,
		}, false)

		if err != nil {
			renderer.JSON(rw, http.StatusBadRequest, map[string]string{
				"status": requests.StatusFailed,
				"error":  err.Error(),
			})
			return
		}
	}

	if err := db.Save(u); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	// Post to slack
	if os.Getenv("ENV") == "production" && !strings.Contains(body.Email, "@bowery.io") {
		channel := "#activity"
		message := u.Name + " " + u.Email + " just signed up."
		username := "Drizzy Drake"
		go slackC.SendMessage(channel, message, username)
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status":    requests.StatusCreated,
		"developer": u,
	})
}

// GET /admin/developers/new, Admin helper for creating developers
func NewDevHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "new", map[string]string{}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// POST /developer/token, logs in a user by creating a new token
func CreateTokenHandler(rw http.ResponseWriter, req *http.Request) {
	var body requests.LoginReq
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	email := body.Email
	password := body.Password
	if email == "" || password == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Email and Password Required.",
		})
		return
	}

	query := map[string]interface{}{"email": email}
	u, err := db.GetDeveloper(query)
	if err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  "No such developer with email " + email + ".",
		})
		return
	}

	if util.HashPassword(password, u.Salt) != u.Password {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Incorrect Password",
		})
		return
	}

	token := util.HashToken()

	update := map[string]interface{}{"token": token}
	if err := db.UpdateDeveloper(query, update); err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status": requests.StatusCreated,
		"token":  token,
	})
}

func CheckAdminHandler(rw http.ResponseWriter, req *http.Request) {
	var body requests.LoginReq
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	email := body.Email
	password := body.Password
	if email == "" || password == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Email and Password Required.",
		})
		return
	}

	query := map[string]interface{}{"email": email}
	u, err := db.GetDeveloper(query)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "not admin",
		})
		return
	}

	if util.HashPassword(password, u.Salt) != u.Password {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "not admin",
		})
		return
	}

	if !u.IsAdmin {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "not admin",
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]string{
		"status": requests.StatusSuccess,
	})
}

// GET /developers/{id}, return public info for a developer
func GetDeveloperByIDHandler(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	token := req.FormValue("token")
	if token == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Valid token required.",
		})
		return
	}

	dev, err := db.GetDeveloperById(id)
	if err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	// If the developer doing the request is not the dev found, only send
	// minimal information.
	if dev.Token != token {
		dev = &schemas.Developer{
			Email:               dev.Email,
			Name:                dev.Name,
			Version:             dev.Version,
			IntegrationEngineer: dev.IntegrationEngineer,
		}
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status":    requests.StatusFound,
		"developer": dev,
	})
}

// GET /developers/me, return the logged in developer
func GetCurrentDeveloperHandler(rw http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	token := req.FormValue("token")
	if token == "" {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  "Valid token required.",
		})
		return
	}

	query := map[string]interface{}{"token": token}
	u, err := db.GetDeveloper(query)
	if err != nil {
		if err == mgo.ErrNotFound {
			err = errors.New("Invalid Token.")
		}

		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status":    requests.StatusFound,
		"developer": u,
	})
}

// POST /session, Creates a new user and charges them for the first year.
func CreateSessionHandler(rw http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	name := req.PostFormValue("name")
	id := req.PostFormValue("id")
	email := req.PostFormValue("stripeEmail")
	if email == "" {
		email = req.PostFormValue("email")
	}

	u := &schemas.Developer{
		Name:       name,
		Email:      email,
		Expiration: time.Now().Add(time.Hour * 24 * 30),
		ID:         bson.ObjectIdHex(id),
	}

	// Silent Signup from cli and not signup form. Will not charge them, but will give them a free month
	if err := db.Save(u); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status":    requests.StatusCreated,
		"developer": u,
	})
}

// POST /developers/{token}/pay payments
func PaymentHandler(rw http.ResponseWriter, req *http.Request) {
	var body requests.PaymentReq
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&body)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	d, err := db.GetDeveloper(map[string]interface{}{"token": mux.Vars(req)["token"]})
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	// Create Stripe Customer
	customerParams := stripe.CustomerParams{
		Email: d.Email,
		Desc:  d.Name,
		Token: body.StripeToken,
	}

	customer, err := stripe.Customers.Create(&customerParams)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	// Charge Stripe Customer
	chargeParams := stripe.ChargeParams{
		Desc:     "Bowery 3",
		Amount:   2900,
		Currency: "usd",
		Customer: customer.Id,
	}

	_, err = stripe.Charges.Create(&chargeParams)
	if err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
		return
	}

	if err := db.UpdateDeveloper(map[string]interface{}{"token": d.Token}, map[string]interface{}{"isPaid": true}); err != nil {
		renderer.JSON(rw, http.StatusInternalServerError, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status":    requests.StatusSuccess,
		"developer": d,
	})
}

// GET /session/{id}, Gets user by ID. If their license has expired it attempts
// to charge them again. It is called everytime crosby is run.
func SessionInfoHandler(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	fmt.Println("Getting user by id", id)
	u, err := db.GetDeveloperById(id)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	if u.Expiration.After(time.Now()) {
		renderer.JSON(rw, http.StatusOK, map[string]interface{}{
			"status":    requests.StatusFound,
			"developer": u,
		})
		return
	}

	if u.StripeToken == "" {
		renderer.JSON(rw, http.StatusOK, map[string]interface{}{
			"status":    requests.StatusExpired,
			"developer": u,
		})
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
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}
	u.Expiration = time.Now()
	if err := db.Save(u); err != nil { // not actually a save, but an update. fix
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status": requests.StatusFound,
		"user":   u,
	})
}

// GET /admin/signup/:id, Renders signup find. Will also handle billing
func SignUpHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "signup", map[string]interface{}{
		"isSignup":     true,
		"stripePubKey": stripePublicKey,
		"id":           mux.Vars(req)["id"],
	}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /admin/thanks!, Renders a thank you/confirmation message stored in static/thanks.html
func ThanksHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "thanks", map[string]interface{}{}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /reset/{email}, Request link to reset password--emails user
func ResetPasswordHandler(rw http.ResponseWriter, req *http.Request) {
	// TODO check empty token
	email := mux.Vars(req)["email"]
	if email == "" {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  "no email provided",
		})
		return
	}

	u, err := db.GetDeveloper(map[string]interface{}{"email": email})
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	message, err := RenderEmail("password_email", map[string]interface{}{
		"name":     strings.Split(u.Name, " ")[0],
		"id":       u.ID.Hex(),
		"token":    u.Token,
		"engineer": u.IntegrationEngineer,
	})
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
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
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]string{
		"status": requests.StatusSuccess,
	})
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
	if err := req.ParseForm(); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	id := req.FormValue("id")
	u, err := db.GetDeveloperById(id)
	if err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	query := map[string]interface{}{"token": mux.Vars(req)["token"]}
	update := map[string]interface{}{"password": util.HashPassword(req.FormValue("new"), u.Salt)}
	if err := db.UpdateDeveloper(query, update); err != nil {
		renderer.JSON(rw, http.StatusBadRequest, map[string]string{
			"status": requests.StatusFailed,
			"error":  err.Error(),
		})
		return
	}

	renderer.JSON(rw, http.StatusOK, map[string]interface{}{
		"status": requests.StatusSuccess,
		"user":   u,
	})
}

// GET /healthz, Indicates that the service is up
func HealthzHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "ok")
}

func StaticHandler(res http.ResponseWriter, req *http.Request) {
	http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))).ServeHTTP(res, req)
}
