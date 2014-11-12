// Copyright 2013-2014 Bowery, Inc.
package db

import (
	"errors"
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/Bowery/gopackages/schemas"
	"github.com/Bowery/gopackages/util"
	"github.com/cenkalti/backoff"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var devs *mgo.Collection

func init() {
	devs = Client.Db.C("developers")
}

func Save(d *schemas.Developer) error {
	if d.Salt == "" {
		d.Salt = uuid.New()
		d.Password = util.HashPassword(d.Password, d.Salt)
	}

	var err error
	b := backoff.NewTicker(backoff.NewExponentialBackOff()).C

	for _ = range b {
		if err = devs.Insert(d); err != nil {
			continue
		}

		break
	}

	return err
}

func GetDeveloper(query bson.M) (*schemas.Developer, error) {
	d := &schemas.Developer{}
	return d, devs.Find(query).One(&d)
}

func GetDeveloperById(id string) (*schemas.Developer, error) {
	return GetDeveloper(bson.M{"_id": bson.ObjectIdHex(id)})
}

func GetDevelopers(query bson.M) ([]*schemas.Developer, error) {
	ds := []*schemas.Developer{}
	return ds, devs.Find(query).All(&ds)
}

func UpdateDeveloper(query, update bson.M) error {
	return devs.Update(query, bson.M{"$set": update})
}

func MockDB() (*schemas.Developer, error) {
	if os.Getenv("ENV") == "production" {
		panic("DON'T RUN MOCKDB IN PRODUCTION!!!!")
		return nil, errors.New("DON't RUN MOCKDB IN PRODUCTION!!!!")
	}
	t, _ := time.Parse(time.RFC3339, "2014-11-10T00:00:00Z")

	dev := &schemas.Developer{
		ID:                  bson.ObjectIdHex("52e7cc4308bcfd732f000028"),
		CreatedAt:           1390922819901,
		Email:               "byrd@bowery.io",
		IsPaid:              false,
		IntegrationEngineer: "David Byrd",
		IsAdmin:             true,
		License:             "660d8268-731d-4cbf-8359-00d23972c4b2",
		Name:                "David Byrd",
		Password:            "64ebf84917bc14112b374c28bb0cdc6fe9941e1aa1681c12519c7f27e967a849",
		Salt:                "a1681ed1-8830-11e3-84be-0d701751111b",
		Token:               "0f0a9ec0-f0e8-11e3-a86e-b9bd016d5ec0",
		Expiration:          t,
	}

	devs.Remove(bson.M{"_id": dev.ID})
	if err := Save(dev); err != nil {
		return nil, err
	}

	return dev, nil
}
