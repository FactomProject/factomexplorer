package main

import (
	"errors"
	"fmt"
	"github.com/FactomProject/factom"
	"log"
	"strings"
)

var DBlocks map[string]*DBlock
var DBlockKeyMRsBySequence map[int]string
var Blocks map[string]*Block
var Entries map[string]*Entry
var Chains map[string]*Chain
var ChainIDsByEncodedName map[string]string
var ChainIDsByDecodedName map[string]string

var BlockIndexes map[string]string //used to index blocks by both their full and partial hash

type DataStatusStruct struct {
	DBlockHeight      int
	LastKnownBlock    string
}

var DataStatus DataStatusStruct

func init() {
	DBlocks = map[string]*DBlock{}
	DBlockKeyMRsBySequence = map[int]string{}
	Blocks = map[string]*Block{}
	Entries = map[string]*Entry{}
	BlockIndexes = map[string]string{}
	Chains = map[string]*Chain{}
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

	AdminBlock       *Block
	FactoidBlock     *Block
	EntryCreditBlock *Block
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

	EntryList []*Entry

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
	FirstEntry *Entry
}

type DecodedString struct {
	Encoded string
	Decoded string
}



func RecordChain(block *Block) {
	if block.PrevBlockHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return
	}

	c:=new(Chain)
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

func StoreEntriesFromBlock(block *Block) {
	for _, v := range block.EntryList {
		Entries[v.Hash] = v
	}
}

//Savers and Loaders
func SaveDBlock(b *DBlock) error {

	return nil
}

func LoadDBlock(hash string) (*DBlock, error) {

	return nil, nil
}

func SaveBlock(b *Block) error {

	return nil
}

func LoadBlock(hash string) (*Block, error) {

	return nil, nil
}

func SaveEntry(b *Entry) error {

	return nil
}

func LoadEntry(hash string) (*Entry, error) {

	return nil, nil
}

func SaveChain(b *Chain) error {

	return nil
}

func LoadChain(hash string) (*Chain, error) {

	return nil, nil
}

func SaveDataStatus(b *DataStatusStruct) error {

	return nil
}

func LoadDataStatus(hash string) (*DataStatusStruct, error) {

	return nil, nil
}


//Getters

func GetBlock(hash string) (*Block, error) {
	hash = strings.ToLower(hash)

	key, ok := BlockIndexes[hash]
	if ok == false {
		return nil, fmt.Errorf("Block %v not found", hash)
	}

	block, ok := Blocks[key]
	if ok == false {
		return nil, fmt.Errorf("Block %v not found", hash)
	}
	return block, nil
}

func GetBlockHeight() int {
	return DataStatus.DBlockHeight
}

func GetDBlocks(start, max int) []*DBlock {
	answer := []*DBlock{}
	for i := start; i <= max; i++ {
		keyMR := DBlockKeyMRsBySequence[i]
		if keyMR == "" {
			continue
		}
		answer = append(answer, DBlocks[keyMR])
	}
	return answer
}

func GetDBlock(keyMR string) (*DBlock, error) {
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



//Getters

func GetEntry(hash string) (*Entry, error) {
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

func GetChains()([]*Chain, error) {
	answer:=[]*Chain{}
	for _, v:=range(Chains) {
		answer = append(answer, v)
	}
	return answer, nil
}

func GetChain(hash string) (*Chain, error) {
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

func GetChainByName(name string) (*Chain, error) {
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
