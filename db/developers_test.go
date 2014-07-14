// Copyright 2014 Bowery, Inc.
package db

import (
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestGetDeveloper(t *testing.T) {
	mock, err := MockDB()
	if err != nil {
		t.Fatal("Unable to Mock DB:", err)
	}

	dev, err := GetDeveloper(bson.M{"email": mock.Email})
	if err != nil {
		t.Fatal("Unable to get developer:", err)
	}

	if dev.ID != mock.ID {
		t.Error("developer not retrieved correctly.")
	}
}

func TestUpdateDeveloper(t *testing.T) {
	mock, err := MockDB()
	if err != nil {
		t.Fatal("Unable to Mock DB:", err)
	}

	testEmail := "testing@email.com"

	if err = UpdateDeveloper(bson.M{"_id": mock.ID}, bson.M{"email": testEmail}); err != nil {
		t.Fatal("Unable to update developer:", err)
	}

	dev, err := GetDeveloperById(mock.ID.Hex())
	if err != nil {
		t.Fatal("Unable to GetDeveloperById:", err)
	}

	if dev.Email != testEmail {
		t.Fatal("UpdateDeveloper did not work")
	}
}

func TestGetDeveloperById(t *testing.T) {
	mock, err := MockDB()
	if err != nil {
		t.Fatal("Unable to Mock DB:", err)
	}

	var id bson.ObjectId

	id = mock.ID

	dev, err := GetDeveloperById(id.Hex())
	if err != nil {
		t.Fatal("Unable to GetDeveloperById:", err)
	}

	if dev.Email != mock.Email {
		t.Error("email not saved correctly.")
	}
}
