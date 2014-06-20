package main

import (
	"encoding/json"
	"github.com/Bowery/broome/db"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
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
	var ok bool
	if token, ok = mock["token"].(string); !ok {
		t.Fatal("Token not a string")
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
