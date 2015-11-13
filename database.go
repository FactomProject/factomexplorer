package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/couchbase/gocb"
	"log"
    //"encoding/json"
)

const DatabaseFile string = "FactomExplorer.db"

var db *bolt.DB

var myCluster *gocb.Cluster
var myBucket *gocb.Bucket
type fullEntry struct {
    DataType string
    DataContent interface{}
}

func Init(filePath string) {
	var err error
	myCluster, err = gocb.Connect("couchbase://localhost")
	if err != nil {
		log.Printf("Error loading myCluster : %+v\n\n", err)
	}
	myBucket, err = myCluster.OpenBucket("default", "")
	if err != nil {
		log.Printf("Error loading myBucket : %+v\n\n", err)
	}
	db, err = bolt.Open(filePath+DatabaseFile, 0600, nil)
	if err != nil {
		panic("Database was not found, and could not be created - " + err.Error())
	}
	for _, v := range BucketList {
		err = db.Update(func(tx *bolt.Tx) error {
		    //fmt.Println("BOLT TX: %s", tx)
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
    //fmt.Println("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ ", dst)
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
	
	//fmt.Printf("sssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssSSS: ", v)
	
	//query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + key + "\" AND DataType=\"" + bucket + "\";")
    //rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    //if qryErr != nil {
    //    fmt.Printf("QUERY ERROR: ", qryErr)
    //}
    //var row interface{}
    //for rows.Next(&row) {
        //jRow, _ := json.Marshal(row)
        //fmt.Printf("UUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU: %+v\n", jRow)
        //fmt.Printf("RRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRow: %+v\n", row)

        //var dat map[string]interface{}
            //if err := json.Unmarshal(jRow, &dat); err != nil {
            //    panic(err)
            //}
            //fmt.Println(dat["default"])

        //dst1 := row.(map[string]interface{})["default"]
        //dType := dst1.(map[string]interface{})["DataType"]
        //dContent := dst1.(map[string]interface{})["DataContent"]
        //dStuff, interCast := dContent.(map[string]interface{})["SequenceNumber"]
        //if interCast != false {
        //    dStuff = dContent.(map[string]string)
        //}
        //fmt.Printf("HHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHH: %+v\n fffffffffffffffffffffffffffffffffffffffffffffffffffffffff %+v\n f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1 %+v\n jjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjj %+v \n", dst1, dType, dContent, dStuff)
        //dd := dContent.(map[string]string)
        //fmt.Println("ADMIN ENTRIES: %+v ........ EntryEntries: %+v ............... Timestamp: %+v", dd["AdminEntries"], dd["EntryEntries"], dd["Timestamp"])
    //}
    //rows.Close()

    //fmt.Printf("AAJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJJ: %+v", v)
	dec := gob.NewDecoder(bytes.NewBuffer(v))
	err = dec.Decode(dst)
	if err != nil {
		log.Printf("Error decoding %v of %v", bucket, key)
		return nil, err
	}
	//fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAa: ", dst)
	
	return dst, nil
}

func SaveData(bucket, key string, toStore interface{}) error {
    //log.Printf("yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy couchbase is trying to saveeeeeeeeeeee: %v ++++ %v +++++ %v ", bucket, key, toStore) 
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
        
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		err := b.Put([]byte(key), data.Bytes())
		return err
	})
	if err != nil {
		log.Printf("Error saving %v of %v - %v", bucket, key, toStore)
		return err
	}
    var value interface{}
    cas, buckErr := myBucket.Get(key, &value)
    if buckErr != nil {
        log.Printf("buckErr::::::::::: ", buckErr)
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
		
    //fmt.Println("Couchbase had this inserted: ") //, couchInsert)
    //fmt.Println(" ............ ", bucket, " ......... ", key)
    //fmt.Println(" ........ %v", toStoreFull)
    //fmt.Println("............................ ", data.Bytes())


	return nil
}
