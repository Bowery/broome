package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"labix.org/v2/mgo/bson"

	"github.com/Bowery/broome/db"
	"github.com/Bowery/broome/requests"
)

var broomeServer http.HandlerFunc = Handler().ServeHTTP

func TestHealthzHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(HealthzHandler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Error("Status Code of Healthz was not 200.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read response body.")
	}

	if string(body) != "ok" {
		t.Error("Healthz body was not ok.")
	}

}

func TestStaticHandler(t *testing.T) {
	testfile := "style.css"
	server := httptest.NewServer(http.HandlerFunc(StaticHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/static/" + testfile)
	if err != nil {
		t.Fatal("Static File request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Error("Status Code of Static File Server was not 200.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read response body.", err)
	}

	expectedBody, err := ioutil.ReadFile("static/" + testfile)
	if err != nil {
		t.Fatal("Unable to read static test file.", err)
	}

	if string(body) != string(expectedBody) {
		t.Error("StaticHandler did not serve the correct file content.")
	}
}

func TestUpdateDeveloperHandler(t *testing.T) {
	mock, err := db.MockDB()
	if err != nil {
		t.Fatal("Could not Mock DB:", err)
	}

	var token string
	if token = mock.Token; token == "" {
		t.Fatal("No token")
	}

	req, err := http.NewRequest("PUT", "http://broome.io/developers/"+token, nil)
	if err != nil {
		t.Fatal("Could not Create Request", err)
	}
	req.SetBasicAuth(token, "")
	req.PostForm = url.Values{
		"name": {"David"},
	}
	res := httptest.NewRecorder()
	broomeServer(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Non-expected status code: %v\tbody: %v", res.Code, res.Body)
	}

	body := map[string]interface{}{}
	if err := json.Unmarshal([]byte(res.Body.String()), &body); err != nil {
		t.Fatal("Response is not valid JSON", err)
	}

	if body["status"] != "updated" {
		t.Fatal("response status should be 'updated' not ", body["status"])
	}

	if reflect.DeepEqual(body["update"], req.PostForm) {
		t.Fatal("response update is not the same as request update", body["update"], req.PostForm)
	}
}

func TestCreateTokenHandler(t *testing.T) {
	_, err := db.MockDB()
	if err != nil {
		t.Fatal("Could not Mock DB:", err)
	}

	var body bytes.Buffer
	bodyReq := map[string]interface{}{"Email": "byrd@bowery.io", "Password": "java$cript"}

	encoder := json.NewEncoder(&body)
	err = encoder.Encode(bodyReq)
	if err != nil {
		t.Fatal("Could not encode JSON:", err)
	}

	req, err := http.NewRequest("POST", "http://broome.io/developers/token", &body)
	if err != nil {
		t.Fatal("Could not create request:", err)
	}
	defer req.Body.Close()

	res := httptest.NewRecorder()
	broomeServer(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Non-expected status code: %v\tbody: %v", res.Code, res.Body)
	}

	resBody := map[string]interface{}{}
	if err := json.Unmarshal([]byte(res.Body.String()), &resBody); err != nil {
		t.Fatal("Response is not valid JSON", err)
	}

	if resBody["status"] != "created" {
		t.Fatal("response status should be 'created' not ", resBody["status"])
	}
}

func TestDeveloperMeHandler(t *testing.T) {
	mock, err := db.MockDB()
	if err != nil {
		t.Fatal("Could not Mock DB:", err)
	}

	var token string
	if token = mock.Token; token == "" {
		t.Fatal("No token.")
	}

	req, err := http.NewRequest("GET", "http://broome.io/developers/me?token="+token, nil)
	if err != nil {
		t.Fatal("Could not Create Request", err)
	}

	res := httptest.NewRecorder()
	broomeServer(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Non-expected status code: %v\tbody: %v", res.Code, res.Body)
	}

	body := &requests.DeveloperRes{}
	if err := json.Unmarshal([]byte(res.Body.String()), body); err != nil {
		t.Fatal("Response is not valid JSON", err)
	}

	if body.Status != "found" {
		t.Fatalf("response status should be 'created' not %v. Error: %v ", body.Status, body.Error)
	}

	var expCreatedAt int64 = 1390922819901

	if body.Developer.CreatedAt != expCreatedAt {
		t.Fatalf("Developer has changed: Date expected %f but got %d", expCreatedAt, body.Developer.CreatedAt)
	}
}

func TestResetRequestHandler(t *testing.T) {
	mock, err := db.MockDB()
	if err != nil {
		t.Fatal("Could not Mock DB:", err)
	}

	var email string
	if email = mock.Email; email == "" {
		t.Fatal("No email")
	}

	req, err := http.NewRequest("GET", "http://broome.io/reset/"+email, nil)
	if err != nil {
		t.Fatal("Could not create request:", err)
	}

	res := httptest.NewRecorder()
	broomeServer(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Non-expected status code: %v\tbody: %v", res.Code, res.Body)
	}

	resBody := map[string]interface{}{}
	if err := json.Unmarshal([]byte(res.Body.String()), &resBody); err != nil {
		t.Fatal("Response is not valid JSON", err)
	}

	if resBody["status"] != "success" {
		t.Fatal("response status should be 'created' not ", resBody["status"])
	}
}

func TestPasswordEditHandler(t *testing.T) {
	mock, err := db.MockDB()
	if err != nil {
		t.Fatal("Could not Mock DB:", err)
	}

	var id bson.ObjectId
	fmt.Printf("%T", mock.ID)
	id = mock.ID

	var token string
	if token = mock.Token; token == "" {
		t.Fatal("Invalid token")
	}

	req, err := http.NewRequest("PUT", "http://broome.io/developers/reset/"+token, nil)
	if err != nil {
		t.Fatal("Could not Create Request", err)
	}
	req.PostForm = url.Values{
		"id":  {id.Hex()},
		"old": {"java$cript"},
		"new": {"password"},
	}
	res := httptest.NewRecorder()
	broomeServer(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Non-expected status code: %v\tbody: %v", res.Code, res.Body)
	}

	body := map[string]interface{}{}
	if err := json.Unmarshal([]byte(res.Body.String()), &body); err != nil {
		t.Fatal("Response is not valid JSON", err)
	}

	if body["status"] != "success" {
		t.Fatal("response status should be 'updated' not ", body["status"])
	}
}
