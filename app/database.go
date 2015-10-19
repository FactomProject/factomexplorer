package app

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"github.com/ThePiachu/Go/Datastore"
	"unicode/utf8"
)

func LoadData(c appengine.Context, bucket, key string, dst interface{}) (interface{}, error) {
	bucket = SanitizeKey(bucket)
	key = SanitizeKey(key)
	err := Datastore.GetFromDatastoreSimpleOrMemcache(c, bucket, key, bucket+key, dst)
	if err == datastore.ErrNoSuchEntity {
		return nil, nil
	}
	return dst, err
}

func SaveData(c appengine.Context, bucket, key string, toStore interface{}) error {
	bucket = SanitizeKey(bucket)
	key = SanitizeKey(key)
	_, err := Datastore.PutInDatastoreSimpleAndMemcache(c, bucket, key, bucket+key, toStore)
	return err
}

func SanitizeKey(key string) string {
	if utf8.Valid([]byte(key)) == false {
		return fmt.Sprintf("%q", key)
	}
	return key
}
