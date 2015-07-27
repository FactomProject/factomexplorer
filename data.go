package main

import (
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
	"log"
	"time"
)

var DBlocks map[string]DBlock
var DBlockKeyMRsBySequence map[int]string
var Blocks map[string]Block

type DataStatusStruct struct {
	DBlockHeight      int
	FullySynchronized bool
}

var DataStatus DataStatusStruct

func init() {
	DBlocks = map[string]DBlock{}
	DBlockKeyMRsBySequence = map[int]string{}
	Blocks = map[string]Block{}
}

type DBlock struct {
	factom.DBlock

	BlockTimeStr string
	KeyMR        string
}

type Block struct {
	ChainID       string
	Hash          string
	PrevBlockHash string

	EntryCount int
	EntryList  []Entry
}

type Entry struct {
	BinaryString string
}

func GetDBlockFromFactom(keyMR string) (DBlock, error) {
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
		block, exists := DBlocks[previousKeyMR]
		if exists {
			if DataStatus.FullySynchronized == true {
				break
			} else {
				previousKeyMR = block.Header.PrevBlockKeyMR
				continue
			}
		}
		body, err := GetDBlockFromFactom(previousKeyMR)
		if err != nil {
			return err
		}
		str, err := EncodeJSONString(body)
		if err != nil {
			return err
		}
		log.Printf("%v", str)

		for _, v := range body.EntryBlockList {
			err = FetchBlock(v.ChainID, v.KeyMR)
			if err != nil {
				return err
			}
		}

		DBlocks[previousKeyMR] = body
		DBlockKeyMRsBySequence[body.Header.SequenceNumber] = previousKeyMR
		if DataStatus.DBlockHeight < body.Header.SequenceNumber {
			DataStatus.DBlockHeight = body.Header.SequenceNumber
		}
		previousKeyMR = body.Header.PrevBlockKeyMR
		if previousKeyMR == "0000000000000000000000000000000000000000000000000000000000000000" {
			DataStatus.FullySynchronized = true
			break
		}

	}
	return nil
}

func FetchBlock(chainID, hash string) error {
	if chainID == "000000000000000000000000000000000000000000000000000000000000000a" ||
		chainID == "000000000000000000000000000000000000000000000000000000000000000c" ||
		chainID == "000000000000000000000000000000000000000000000000000000000000000f" {

		raw, err := factom.GetRaw(hash)
		if err != nil {
			return err
		}
		var block Block
		switch chainID {
		case "000000000000000000000000000000000000000000000000000000000000000a":
			block, err = ParseAdminBlock(chainID, hash, raw)
			if err != nil {
				return err
			}
			break
		case "000000000000000000000000000000000000000000000000000000000000000c":

			break
		case "000000000000000000000000000000000000000000000000000000000000000f":

			break
		}
		Blocks[hash] = block

	} else {
		_, err := factom.GetEBlock(hash)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseAdminBlock(chainID, hash string, rawBlock []byte) (Block, error) {
	var answer Block

	aBlock := new(common.AdminBlock)
	_, err := aBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return answer, err
	}

	answer.ChainID = chainID
	answer.Hash = hash
	answer.EntryCount = len(aBlock.ABEntries)
	answer.PrevBlockHash = fmt.Sprintf("%X", aBlock.Header.PrevFullHash.GetBytes())
	/*answer.EntryList = make([]Entry, answer.EntryCount)
	for i, v := range aBlock.ABEntries {
		marshalled, err := v.MarshalBinary()
		if err != nil {
			return answer, err
		}
		answer.EntryList[i].BinaryString = fmt.Sprintf("%X", marshalled)
	}*/

	return answer, nil
}

func GetBlockHeight() int {
	return DataStatus.DBlockHeight
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

func GetDBlock(keyMR string) (DBlock, error) {
	block, ok := DBlocks[keyMR]
	if ok != true {
		return block, errors.New("DBlock not found")
	}
	return block, nil
}

type DBInfo struct {
	BTCTxHash string
}

func GetDBInfo(keyMR string) (DBInfo, error) {
	//TODO: gather DBInfo
	return DBInfo{}, nil
}

type EBlock struct {
	factom.EBlock
}
