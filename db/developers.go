// Copyright 2013-2014 Bowery, Inc.
package db

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/Bowery/broome/util"
	"github.com/cenkalti/backoff"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"os"
	"time"
)

var (
	developers *mgo.Collection
)

type Developer struct {
	ID                  bson.ObjectId `bson:"_id,omitempty"json:"_id,omitempty"`
	Name                string        `bson:"name,omitempty"json:"name,omitempty"`
	Email               string        `bson:"email,omitempty"json:"email,omitempty"`
	Password            string        `bson:"password,omitempty"json:"password,omitempty"`
	Salt                string        `bson:"salt,omitempty"json:"salt,omitempty"`
	Token               string        `bson:"token,omitempty"json:"token,omitempty"`
	IsAdmin             bool          `bson:"isAdmin,omitempty"json:"isAdmin,omitempty"`
	StripeToken         string        `bson:"stripeToken,omitempty"json:"stripeToken,omitempty"`
	NextPaymentTime     time.Time     `bson:"nextPaymentTime,omitempty"json:"nextPaymentTime,omitempty"`
	IntegrationEngineer string        `bson:"integrationEngineer,omitempty"json:"integrationEngineer,omitempty"`
	CreatedAt           time.Time     `bson:"createdAt,omitempty"json:"createdAt,omitempty"`
	LastActiveAt        time.Time     `bson:"lastActiveAt,omitempty"json:"lastActiveAt,omitempty"`
}

func init() {
	dbAddr := "localhost"
	if os.Getenv("ENV") == "production" {
		dbAddr = "ec2-54-196-181-224.compute-1.amazonaws.com"
	}

	session, err := mgo.Dial(dbAddr)
	if err != nil {
		panic(err)
	}

	db := session.DB("bowery")

	if os.Getenv("ENV") == "production" {
		if err := db.Login("bowery", "java$cript"); err != nil {
			panic(err)
		}
	}
	developers = db.C("developers")
}

func (d *Developer) Save() error {
	if d.Salt == "" {
		d.Salt = uuid.New()
		d.Password = util.HashPassword(d.Password, d.Salt)
	}

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

func GetDevelopers(query bson.M) ([]*Developer, error) {
	out := []*Developer{}
	return out, developers.Find(query).All(&out)
}

func GetDeveloper(query bson.M) (*Developer, error) {
	d := &Developer{}
	return d, developers.Find(query).One(d)
}
