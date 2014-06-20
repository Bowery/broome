// Copyright 2014 Bowery, Inc.
package db

import (
	"fmt"
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestGetDeveloperById(t *testing.T) {
	mock, err := MockDB()
	if err != nil {
		t.Fatal("Unable to Mock DB:", err)
	}

	var id bson.ObjectId
	var ok bool
	if id, ok = mock["_id"].(bson.ObjectId); !ok {
		t.Fatal("Unable to cast mock _id.")
	}

	fmt.Println(id.Hex())
	dev, err := GetDeveloperById(id.Hex())
	if err != nil {
		t.Fatal("Unable to GetDeveloperById:", err)
	}

	if dev.Email != mock["email"] {
		t.Error("email not saved correctly.")
	}
}
