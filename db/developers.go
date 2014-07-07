// Copyright 2013-2014 Bowery, Inc.
package db

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
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
	Password            string        `bson:"password,omitempty"json:"-,omitempty"`
	Salt                string        `bson:"salt,omitempty"json:"-,omitempty"`
	Token               string        `bson:"token,omitempty"json:"token,omitempty"`
	IsAdmin             bool          `bson:"isAdmin,omitempty"json:"isAdmin,omitempty"`
	IsPaid              bool          `bson"isPaid,omitempty"json:"isPaid,omitempty"`
	StripeToken         string        `bson:"stripeToken,omitempty"json:"stripeToken,omitempty"`
	NextPaymentTime     time.Time     `bson:"nextPaymentTime,omitempty"json:"nextPaymentTime,omitempty"`
	IntegrationEngineer string        `bson:"integrationEngineer,omitempty"json:"integrationEngineer,omitempty"`
	CreatedAt           int64         `bson:"createdAt,omitempty"json:"createdAt,omitempty"`
	LastActiveAt        time.Time     `bson:"lastActiveAt,omitempty"json:"lastActiveAt,omitempty"`
}

func init() {
	dbAddr := "localhost"

	if os.Getenv("ENV") == "development" {
		dbAddr = "10.0.0.15:27017"
	}

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
	return d, developers.FindId(bson.ObjectIdHex(id)).One(&d)
}

func GetDevelopers(query bson.M) ([]*Developer, error) {
	out := []*Developer{}
	return out, developers.Find(query).All(&out)
}

func GetDeveloper(query bson.M) (*Developer, error) {
	d := &Developer{}
	return d, developers.Find(query).One(d)
}

func UpdateDeveloper(query, update bson.M) error {
	return developers.Update(query, bson.M{"$set": update})
}

func MockDB() (bson.M, error) {
	if os.Getenv("ENV") == "production" {
		panic("DON'T RUN MOCKDB IN PRODUCTION!!!!")
		return nil, errors.New("DON't RUN MOCKDB IN PRODUCTION!!!!")
	}
	t, _ := time.Parse(time.RFC3339, "2014-11-10T00:00:00Z")

	dev := bson.M{
		"_id":                 bson.ObjectIdHex("52e7cc4308bcfd732f000028"),
		"createdAt":           1390922819901,
		"email":               "byrd@bowery.io",
		"integrationEngineer": "David Byrd",
		"isAdmin":             true,
		"isConnected":         false,
		"isPaid":              false,
		"lastActive":          1402967272406,
		"license":             "660d8268-731d-4cbf-8359-00d23972c4b2",
		"name":                "David Byrd",
		"password":            "64ebf84917bc14112b374c28bb0cdc6fe9941e1aa1681c12519c7f27e967a849",
		"salt":                "a1681ed1-8830-11e3-84be-0d701751111b",
		"token":               "0f0a9ec0-f0e8-11e3-a86e-b9bd016d5ec0",
		"nextPaymentTime":     t,
	}

	developers.Remove(bson.M{"_id": dev["_id"]}) // ignore potential 'not found' error
	if err := developers.Insert(dev); err != nil {
		return nil, err
	}

	return dev, nil
}
