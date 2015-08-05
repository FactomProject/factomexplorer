package main

import (
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
	"github.com/FactomProject/factom"
	"log"
	"strings"
	"time"
)

var DBlocks map[string]DBlock
var DBlockKeyMRsBySequence map[int]string
var Blocks map[string]Block
var Entries map[string]Entry
var Chains map[string]Chain
var ChainIDsByEncodedName map[string]string
var ChainIDsByDecodedName map[string]string

var BlockIndexes map[string]string //used to index blocks by both their full and partial hash

type DataStatusStruct struct {
	DBlockHeight      int
	FullySynchronized bool
	LastKnownBlock    string
}

var DataStatus DataStatusStruct

func init() {
	DBlocks = map[string]DBlock{}
	DBlockKeyMRsBySequence = map[int]string{}
	Blocks = map[string]Block{}
	Entries = map[string]Entry{}
	BlockIndexes = map[string]string{}
	Chains = map[string]Chain{}
	ChainIDsByEncodedName = map[string]string{}
	ChainIDsByDecodedName = map[string]string{}

	DataStatus.LastKnownBlock = "0000000000000000000000000000000000000000000000000000000000000000"
}

type DBlock struct {
	factom.DBlock

	BlockTimeStr string
	KeyMR        string

	Blocks int

	AdminEntries       int
	EntryCreditEntries int
	FactoidEntries     int
	EntryEntries       int

	AdminBlock       Block
	FactoidBlock     Block
	EntryCreditBlock Block
}

type Common struct {
	ChainID   string
	Timestamp string

	JSONString   string
	SpewString   string
	BinaryString string
}

type Block struct {
	Common

	FullHash    string //KeyMR
	PartialHash string

	PrevBlockHash string

	EntryCount int

	EntryList []Entry

	IsAdminBlock       bool
	IsFactoidBlock     bool
	IsEntryCreditBlock bool
	IsEntryBlock       bool
}

type Entry struct {
	Common

	ExternalIDs []DecodedString
	Content DecodedString

	//Marshallable blocks
	Hash string
}

type Chain struct {
	ChainID string
	Names []DecodedString
	FirstEntryID string

	//Not saved
	FirstEntry Entry
}

type DecodedString struct {
	Encoded string
	Decoded string
}

func GetDBlockFromFactom(keyMR string) (DBlock, error) {
	var answer DBlock

	body, err := factom.GetDBlock(keyMR)
	if err != nil {
		return answer, err
	}

	answer = DBlock{DBlock: *body}
	answer.BlockTimeStr = TimestampToString(body.Header.TimeStamp)
	answer.KeyMR = keyMR

	return answer, nil
}

func TimestampToString(timestamp uint64) string {
	blockTime := time.Unix(int64(timestamp), 0)
	return blockTime.Format("2006-01-02 15:04:05")
}

func Synchronize() error {
	log.Println("Synchronize()")
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

		log.Printf("\n\nProcessing block number %v\n\n", body.Header.SequenceNumber)

		str, err := EncodeJSONString(body)
		if err != nil {
			return err
		}
		log.Printf("%v", str)

		for _, v := range body.EntryBlockList {
			fetchedBlock, err := FetchBlock(v.ChainID, v.KeyMR, body.BlockTimeStr)
			if err != nil {
				return err
			}
			switch v.ChainID {
			case "000000000000000000000000000000000000000000000000000000000000000a":
				body.AdminEntries += fetchedBlock.EntryCount
				body.AdminBlock = fetchedBlock
				break
			case "000000000000000000000000000000000000000000000000000000000000000c":
				body.EntryCreditEntries += fetchedBlock.EntryCount
				body.EntryCreditBlock = fetchedBlock
				break
			case "000000000000000000000000000000000000000000000000000000000000000f":
				body.FactoidEntries += fetchedBlock.EntryCount
				body.FactoidBlock = fetchedBlock
				break
			default:
				body.EntryEntries += fetchedBlock.EntryCount
				break
			}
		}
		body.EntryBlockList = body.EntryBlockList[3:]

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

func FetchBlock(chainID, hash, blockTime string) (Block, error) {
	var block Block

	raw, err := factom.GetRaw(hash)
	if err != nil {
		return block, err
	}
	switch chainID {
	case "000000000000000000000000000000000000000000000000000000000000000a":
		block, err = ParseAdminBlock(chainID, hash, raw, blockTime)
		if err != nil {
			return block, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000c":
		block, err = ParseEntryCreditBlock(chainID, hash, raw, blockTime)
		if err != nil {
			return block, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000f":
		block, err = ParseFactoidBlock(chainID, hash, raw, blockTime)
		if err != nil {
			return block, err
		}
		break
	default:
		block, err = ParseEntryBlock(chainID, hash, raw, blockTime)
		if err != nil {
			return block, err
		}
		break
	}

	StoreEntriesFromBlock(block)
	Blocks[hash] = block

	BlockIndexes[block.FullHash] = hash
	BlockIndexes[block.PartialHash] = hash

	if block.IsEntryBlock {
		RecordChain(block)
	}

	return block, nil
}

func RecordChain(block Block) {
	if block.PrevBlockHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return
	}

	var c Chain
	c.ChainID = block.ChainID
	c.FirstEntryID = block.EntryList[0].Hash
	c.Names = block.EntryList[0].ExternalIDs[:]

	Chains[c.ChainID] = c
	for _, v:=range(c.Names) {
		ChainIDsByDecodedName[v.Decoded] = c.ChainID
		ChainIDsByEncodedName[v.Encoded] = c.ChainID
	}

	log.Printf("\n\nChain - %v\n\n", c)
}

func StoreEntriesFromBlock(block Block) {
	for _, v := range block.EntryList {
		Entries[v.Hash] = v
	}
}

func ParseEntryCreditBlock(chainID, hash string, rawBlock []byte, blockTime string) (Block, error) {
	var answer Block

	ecBlock := common.NewECBlock()
	_, err := ecBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return answer, err
	}

	answer.ChainID = chainID
	answer.FullHash = ecBlock.Hash().String()
	answer.PartialHash = ecBlock.HeaderHash().String()
	answer.PrevBlockHash = ecBlock.Header.PrevLedgerKeyMR.String()

	answer.EntryCount = len(ecBlock.Body.Entries)
	answer.EntryList = make([]Entry, answer.EntryCount)

	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	for i, v := range ecBlock.Body.Entries {
		var entry Entry

		marshalled, err := v.MarshalBinary()
		if err != nil {
			return answer, err
		}
		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return answer, err
		}
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}

	answer.JSONString, err = ecBlock.JSONString()
	if err != nil {
		return answer, err
	}
	answer.SpewString = ecBlock.Spew()

	answer.IsEntryCreditBlock = true

	return answer, nil
}

func ParseFactoidBlock(chainID, hash string, rawBlock []byte, blockTime string) (Block, error) {
	var answer Block

	fBlock := new(block.FBlock)
	_, err := fBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return answer, nil
	}

	answer.ChainID = chainID
	answer.PartialHash = hash
	answer.PrevBlockHash = fmt.Sprintf("%x", fBlock.PrevKeyMR.Bytes())

	transactions := fBlock.GetTransactions()
	answer.EntryCount = len(transactions)
	answer.EntryList = make([]Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	for i, v := range transactions {
		var entry Entry
		bin, err:=v.MarshalBinary()

		if err != nil {
			return answer, nil
		}


		entry.BinaryString = fmt.Sprintf("%x", bin)
		entry.Timestamp = TimestampToString(v.GetMilliTimestamp() / 1000)
		entry.Hash = v.GetHash().String()
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return answer, err
		}
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = fBlock.JSONString()
	if err != nil {
		return answer, err
	}
	answer.SpewString = fBlock.Spew()

	answer.IsFactoidBlock = true

	return answer, nil
}

func ParseEntryBlock(chainID, hash string, rawBlock []byte, blockTime string) (Block, error) {
	var answer Block

	eBlock := common.NewEBlock()
	_, err := eBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return answer, err
	}

	answer.ChainID = chainID
	answer.PartialHash = eBlock.KeyMR().String()
	answer.FullHash = eBlock.Hash().String()

	answer.PrevBlockHash = eBlock.Header.PrevKeyMR.String()

	answer.EntryCount = len(eBlock.Body.EBEntries)
	answer.EntryList = make([]Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	for i, v := range eBlock.Body.EBEntries {
		entry, err:=FetchAndParseEntry(v.String(), blockTime)
		if err != nil {
			return answer, err
		}

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = eBlock.JSONString()
	if err != nil {
		return answer, err
	}
	answer.SpewString = eBlock.Spew()

	answer.IsEntryBlock = true

	return answer, nil
}

func FetchAndParseEntry(hash, blockTime string) (Entry, error) {
	var e Entry
	raw, err:=factom.GetRaw(hash)
	if err!=nil {
		return e, err
	}

	entry:=new(common.Entry)
	_, err = entry.UnmarshalBinaryData(raw)
	if err!=nil {
		return e, err
	}


	e.ChainID = entry.ChainID.String()
	e.Hash = hash
	str, err:=entry.JSONString()
	if err!=nil {
		return e, err
	}
	e.JSONString = str 
	e.SpewString = entry.Spew()
	e.BinaryString = fmt.Sprintf("%x", raw)
	e.Timestamp = blockTime

	e.Content = ByteSliceToDecodedString(entry.Content)
	e.ExternalIDs = make([]DecodedString, len(entry.ExtIDs))
	for i, v:=range(entry.ExtIDs) {
		e.ExternalIDs[i] = ByteSliceToDecodedString(v)
	}

	log.Printf("\n\n Entry - %v\n\n", e)

	Entries[hash] = e

	return e, nil
}

func ByteSliceToDecodedString(b []byte) (DecodedString) {
	var ds DecodedString
	ds.Encoded = fmt.Sprintf("%x", b)
	ds.Decoded = string(b)
	return ds
}

func ParseAdminBlock(chainID, hash string, rawBlock []byte, blockTime string) (Block, error) {
	var answer Block

	aBlock := new(common.AdminBlock)
	_, err := aBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return answer, err
	}

	answer.ChainID = chainID
	fullHash, err := aBlock.LedgerKeyMR()
	if err != nil {
		return answer, err
	}
	answer.FullHash = fullHash.String()
	partialHash, err := aBlock.PartialHash()
	if err != nil {
		return answer, err
	}
	answer.PartialHash = partialHash.String()
	answer.EntryCount = len(aBlock.ABEntries)
	answer.PrevBlockHash = fmt.Sprintf("%x", aBlock.Header.PrevLedgerKeyMR.GetBytes())
	answer.EntryList = make([]Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	for i, v := range aBlock.ABEntries {
		marshalled, err := v.MarshalBinary()
		if err != nil {
			return answer, err
		}
		var entry Entry

		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Hash = fmt.Sprintf("%x", marshalled)
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return answer, err
		}
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = aBlock.JSONString()
	if err != nil {
		return answer, err
	}
	answer.SpewString = aBlock.Spew()

	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	answer.IsAdminBlock = true

	return answer, nil
}

//Getters

func GetBlock(hash string) (Block, error) {
	hash = strings.ToLower(hash)
	var block Block
	key, ok := BlockIndexes[hash]
	if ok == false {
		return block, fmt.Errorf("Block %v not found", hash)
	}

	block, ok = Blocks[key]
	if ok == false {
		return block, fmt.Errorf("Block %v not found", hash)
	}
	return block, nil
}

func GetBlockHeight() int {
	return DataStatus.DBlockHeight
}

func GetDBlocks(start, max int) []DBlock {
	answer := []DBlock{}
	for i := start; i <= max; i++ {
		keyMR := DBlockKeyMRsBySequence[i]
		if keyMR == "" {
			continue
		}
		answer = append(answer, DBlocks[keyMR])
	}
	return answer
}

func GetDBlock(keyMR string) (DBlock, error) {
	keyMR = strings.ToLower(keyMR)
	block, ok := DBlocks[keyMR]
	if ok != true {
		return block, errors.New("DBlock not found")
	}
	return block, nil
}

type DBInfo struct {
	BTCTxHash string
}

func GetEntry(hash string) (Entry, error) {
	hash = strings.ToLower(hash)
	entry, ok := Entries[hash]
	if ok != true {
		str, _ := EncodeJSONString(Entries)
		log.Printf("%v not found in %v", hash, str)
		return entry, errors.New("Entry not found")
	}
	return entry, nil
}

func GetDBInfo(keyMR string) (DBInfo, error) {
	//TODO: gather DBInfo
	return DBInfo{}, nil
}

func GetChains()([]Chain, error) {
	answer:=[]Chain{}
	for _, v:=range(Chains) {
		answer = append(answer, v)
	}
	return answer, nil
}

func GetChain(hash string) (Chain, error) {
	hash = strings.ToLower(hash)
	chain, found := Chains[hash]
	if found == false {
		return chain, errors.New("Chain not found")
	}
	entry, found := Entries[chain.FirstEntryID]
	if found == false {
		return chain, errors.New("First entry not found")
	}
	chain.FirstEntry = entry
	return chain, nil
}

func GetChainByName(name string) (Chain, error) {
	id, found:=ChainIDsByEncodedName[name]
	if found == false {
		id, found = ChainIDsByDecodedName[name]
		if found == false {
			return GetChain(name)
		}
	}
	return GetChain(id)
}

type EBlock struct {
	factom.EBlock
}
