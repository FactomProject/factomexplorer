package app

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ThePiachu/Go/Datastore"
	"github.com/ThePiachu/Go/Log"
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
	if err != nil {
		Log.Errorf(c, "SaveData - Error saving %v from %v - %v", key, bucket, err)
		return err
	}
	return nil
}

func SaveBlockData(c appengine.Context, bucket, key string, b *Block) error {
	if b.InBlobstore {
		//Blobkey appengine.BlobKey
		err := Datastore.PutInFakeBlobstore(c, bucket, key, b)
		if err != nil {
			c.Errorf("SaveBlockData - %v", err)
			return err
		}

		b.TrimData()
		str, _ := b.JSON()
		Log.Infof(c, "Trimmed data - %v", str)
	}

	bucket = SanitizeKey(bucket)
	key = SanitizeKey(key)
	_, err := Datastore.PutInDatastoreSimpleAndMemcache(c, bucket, key, bucket+key, b)

	if err != nil {
		if strings.Contains(err.Error(), "server returned the wrong number of keys") ||
			strings.Contains(err.Error(), "Limit for datastore_v3 is") {
			//Block is too big to put into datastore, we need to use blobstore
			Log.Warningf(c, "Putting %v, %v in Blobstore", key, bucket)
			b.InBlobstore = true
			return SaveBlockData(c, bucket, key, b)
		} else {
			c.Errorf("SaveBlockData - %v", err)
			return err
		}
	}
	return nil
}

func LoadBlockData(c appengine.Context, bucket, key string, dst *Block) (interface{}, error) {
	bucket = SanitizeKey(bucket)
	key = SanitizeKey(key)
	err := Datastore.GetFromDatastoreSimpleOrMemcache(c, bucket, key, bucket+key, dst)
	if err == datastore.ErrNoSuchEntity {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if dst.InBlobstore {
		err = Datastore.GetFromFakeBlobstore(c, bucket, key, dst)
		if err != nil {
			c.Errorf("LoadBlockData - %v, ", err)
			return nil, err
		}
	}

	return dst, err
}

func SanitizeKey(key string) string {
	if utf8.Valid([]byte(key)) == false {
		return fmt.Sprintf("%q", key)
	}
	return key
}
