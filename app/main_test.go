package app_test

import (
	"appengine/aetest"
	"fmt"
	"github.com/FactomProject/factom"
	"github.com/ThePiachu/Go/Datastore"
	"github.com/ThePiachu/Go/mymath"
	"testing"
	"unicode/utf8"
)

/*
func Test(t *testing.T) {
	GetAddressInformationFromFactom("FA3eNd17NgaXZA3rXQVvzSvWHrpXfHWPzLQjJy2PQVQSc4ZutjC1")
	//GetAddressInformationFromFactom("dceb1ce5778444e7777172e1f586488d2382fb1037887cd79a70b0cba4fb3dce")
	t.Errorf("Test")
}
*/

func GetAddressInformationFromFactom(address string) error {
	ecBalance, err := factom.ECBalance(address)
	if err != nil {
		fmt.Printf("GetAddressInformationFromFactom 1 - %v\n", err)
	} else {
		fmt.Printf("ECBalance - %v\n", ecBalance)
	}
	fctBalance, err := factom.FctBalance(address)
	if err != nil {
		fmt.Printf("GetAddressInformationFromFactom 2 - %v\n", err)
	} else {
		fmt.Printf("FctBalance - %v\n", fctBalance)
	}

	return nil
}

type Test struct {
	Test string
}

func testMemcache(t *testing.T) {
	hexStr := "ca81e518e9a5519b7b218b85b13d73447f65c48c9c6f1b67db55a54ab48fc1de"
	hex := mymath.Str2Hex(hexStr)
	str := mymath.Hex2ASCII(hex)
	fmt.Printf("Test - %v\n", str)

	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Error(err)
	}

	defer c.Close()

	toPut := new(Test)
	toPut.Test = "test"

	err = Datastore.PutInMemcache(c, str, toPut)
	if err != nil {
		t.Error(err)
	}

	_, err = Datastore.PutInDatastoreSimpleAndMemcache(c, "test", str, str, toPut)
	if err != nil {
		t.Error(err)
	}
}

func TestUFT8(t *testing.T) {
	hexStr := "ca81e518e9a5519b7b218b85b13d73447f65c48c9c6f1b67db55a54ab48fc1de"
	hex := mymath.Str2Hex(hexStr)
	str := mymath.Hex2ASCII(hex)
	fmt.Printf("Test - %v\n", str)

	fmt.Println(utf8.Valid(hex))
	str2 := SanitizeKey(str)
	fmt.Printf("Test2 - %v\n", str2)
	str3 := SanitizeKey("This is plaintext, nothing to see here")
	fmt.Printf("Test2 - %v\n", str3)

}

func SanitizeKey(key string) string {
	if utf8.Valid([]byte(key)) == false {
		return fmt.Sprintf("%q", key)
	}
	return key
}
