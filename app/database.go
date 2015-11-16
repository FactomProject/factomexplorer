package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unicode/utf8"
	"github.com/couchbase/gocb"
	"log"
)

var myCluster *gocb.Cluster
var myBucket *gocb.Bucket
type fullEntry struct {
    DataType string
    DataContent interface{}
}

func OriginalInit() {
	var err error
	myCluster, err = gocb.Connect("couchbase://localhost")
	if err != nil {
		log.Printf("Error loading myCluster : %+v\n\n", err)
	}
	myBucket, err = myCluster.OpenBucket("default", "")
	if err != nil {
	}
}
		
		
func LoadData(bucket, key string, dst interface{}) (interface{}, error) {
	if cfg.UseDatabase == false {
		return nil, nil
	}
	return dst, nil
}

func SaveData(bucket, key string, toStore interface{}) error {
    key = SanitizeKey(key)
	if cfg.UseDatabase == false {
		return nil
	}
	var data bytes.Buffer

	enc := gob.NewEncoder(&data)

	err := enc.Encode(toStore)
	if err != nil {
		return err
	}

	toStoreFull := fullEntry{ 
            DataType: bucket,
            DataContent: toStore,
        }

    var value interface{}
	
	_, err = myBucket.Insert(key, toStoreFull, 0)
	if err != nil {
        cas, buckErr := myBucket.Get(key, &value)
        if buckErr != nil {
            if buckErr.Error() != "Key not found." {
                log.Printf("buckErr::::::::::: ", buckErr)
            }
        } else {
            toStoreFull = fullEntry{
                DataType: bucket,
                DataContent: toStore,
            }
            _, delErr := myBucket.Remove(key, cas)
            if delErr != nil {
                log.Printf("delErr :::::::::::::::: ", delErr)
            }
        }
	    _, err = myBucket.Insert(key, toStoreFull, 0)
	    if err != nil {
		    log.Printf("Error in couchbase saving %v of %v - %v", bucket, key, toStore)
		    return err
        }
	}


	return nil
}


func SanitizeKey(key string) string {
	if utf8.Valid([]byte(key)) == false {
		return fmt.Sprintf("%q", key)
	}
	return key
}
