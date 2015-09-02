package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

const DatabaseFile string = "FactomExplorer.db"

var db *bolt.DB

func Init(filePath string) {
	var err error
	db, err = bolt.Open(filePath+DatabaseFile, 0600, nil)
	if err != nil {
		panic("Database was not found, and could not be created - " + err.Error())
	}
	for _, v := range BucketList {
		err = db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(v))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
}

func LoadData(bucket, key string, dst interface{}) (interface{}, error) {
	if cfg.UseDatabase == false {
		return nil, nil
	}

	var v []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v1 := b.Get([]byte(key))
		if v1 == nil {
			return nil
		}
		v = make([]byte, len(v1))
		copy(v, v1)
		return nil
	})
	if err != nil {
		log.Printf("Error loading %v of %v", bucket, key)
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	dec := gob.NewDecoder(bytes.NewBuffer(v))
	err = dec.Decode(dst)
	if err != nil {
		log.Printf("Error decoding %v of %v", bucket, key)
		return nil, err
	}

	return dst, nil
}

func SaveData(bucket, key string, toStore interface{}) error {
	if cfg.UseDatabase == false {
		return nil
	}

	var data bytes.Buffer

	enc := gob.NewEncoder(&data)

	err := enc.Encode(toStore)
	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		err := b.Put([]byte(key), data.Bytes())
		return err
	})
	if err != nil {
		log.Printf("Error saving %v of %v - %v", bucket, key, toStore)
		return err
	}

	return nil
}
