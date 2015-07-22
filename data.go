package main

import (
	"github.com/FactomProject/factom"
	"log"
	"time"
)

var DBlocks map[string]DBlock
var DBlockKeyMRsBySequence map[int]string
var DBlockHeight int

func init() {
	DBlocks = map[string]DBlock{}
	DBlockKeyMRsBySequence = map[int]string{}
}

type DBlock struct {
	factom.DBlock

	BlockTimeStr string
	KeyMR        string
}

func GetDBlock(keyMR string) (DBlock, error) {
	var answer DBlock

	body, err := factom.GetDBlock(keyMR)
	if err != nil {
		return answer, err
	}

	answer = DBlock{DBlock: *body}
	blockTime := time.Unix(int64(body.Header.TimeStamp), 0)
	answer.BlockTimeStr = blockTime.Format("2006-01-02 15:04:05")
	answer.KeyMR = keyMR

	return answer, nil
}

func Synchronize() error {
	head, err := factom.GetDBlockHead()
	if err != nil {
		return err
	}
	previousKeyMR := head.KeyMR
	for {
		body, err := GetDBlock(previousKeyMR)
		if err != nil {
			return err
		}
		str, err := EncodeJSONString(body)
		if err != nil {
			return err
		}
		log.Printf("%v\n", str)
		DBlocks[previousKeyMR] = body
		DBlockKeyMRsBySequence[body.Header.SequenceNumber] = previousKeyMR
		if DBlockHeight < body.Header.SequenceNumber {
			DBlockHeight = body.Header.SequenceNumber
		}
		previousKeyMR = body.Header.PrevBlockKeyMR
		if previousKeyMR == "0000000000000000000000000000000000000000000000000000000000000000" {
			break
		}

		_, exists := DBlocks[previousKeyMR]
		if exists {
			break
		}
	}
	return nil
}

func GetBlockHeight() int {
	return DBlockHeight
}

func GetDBlocks(start, max int) []DBlock {
	answer := []DBlock{}
	for i := start; i < max; i++ {
		keyMR := DBlockKeyMRsBySequence[i]
		if keyMR == "" {
			continue
		}
		answer = append(answer, DBlocks[keyMR])
	}
	return answer
}
