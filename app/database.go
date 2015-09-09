package main

import (
	"appengine"
	"github.com/ThePiachu/Go/Datastore"
)

func LoadData(c appengine.Context, bucket, key string, dst interface{}) (interface{}, error) {
	err := Datastore.GetFromDatastoreSimpleOrMemcache(c, bucket, key, bucket+key, dst)
	return dst, err
}

func SaveData(c appengine.Context, bucket, key string, toStore interface{}) error {
	_, err := Datastore.PutInDatastoreSimpleAndMemcache(c, bucket, key, bucket+key, toStore)
	return err
}
