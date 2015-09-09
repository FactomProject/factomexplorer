package app_test

import (
	"fmt"
	"github.com/FactomProject/factom"
	"testing"
)

func Test(t *testing.T) {
	GetAddressInformationFromFactom("FA3eNd17NgaXZA3rXQVvzSvWHrpXfHWPzLQjJy2PQVQSc4ZutjC1")
	//GetAddressInformationFromFactom("dceb1ce5778444e7777172e1f586488d2382fb1037887cd79a70b0cba4fb3dce")
	t.Errorf("Test")
}

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
