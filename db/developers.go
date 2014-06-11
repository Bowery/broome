// Copyright 2013-2014 Bowery, Inc.
package db

import (
	"encoding/json"
	"github.com/cenkalti/backoff"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

var (
	developers *mgo.Collection
)

type Developer struct {
	ID                  string    `bson:"_id,omitempty"json:"_id,omitempty"`
	Name                string    `bson:"name,omitempty"json:"name,omitempty"`
	Email               string    `bson:"email,omitempty"json:"email,omitempty"`
	Password            string    `bson:"password,omitempty"json:"password,omitempty"`
	Salt                string    `bson:"salt,omitempty"json:"salt,omitempty"`
	Token               string    `bson:"token,omitempty"json:"token,omitempty"`
	IsAdmin             bool      `bson:"isAdmin,omitempty"json:"isAdmin,omitempty"`
	StripeToken         string    `bson:"stripeToken,omitempty"json:"stripeToken,omitempty"`
	NextPaymentTime     time.Time `bson:"nextPaymentTime,omitempty"json:"nextPaymentTime,omitempty"`
	IntegrationEngineer string    `bson:"integrationEngineer,omitempty"json:"integrationEngineer,omitempty"`
	CreatedAt           time.Time `bson:"createdAt,omitempty"json:"createdAt,omitempty"`
	LastActiveAt        time.Time `bson:"lastActiveAt,omitempty"json:"lastActiveAt,omitempty"`
}

func init() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	developers = session.DB("bowery").C("developers")
}

func (d *Developer) Save() error {
	var err error
	b := backoff.NewTicker(backoff.NewExponentialBackoff()).C
	for _ = range b {
		if err = developers.Insert(d); err != nil {
			continue
		}

		break
	}

	return err
}

func GetDeveloperById(id string) (*Developer, error) {
	d := &Developer{}
	return d, developers.FindId(id).One(&d)
}

func GetDeveloper(d *Developer) (*Developer, error) {
	query := bson.M{}

	raw, err := json.Marshal(d)
	if err != nil {
		return d, err
	}

	if err := json.Unmarshal(raw, query); err != nil {
		return d, err
	}

	return d, developers.Find(query).One(d)
}
