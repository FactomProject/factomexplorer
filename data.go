package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
	"log"
	"strings"
)

var DBlocks map[string]*DBlock
var DBlockKeyMRsBySequence map[string]string
var Blocks map[string]*Block
var Entries map[string]*Entry
var Chains map[string]*Chain
var ChainIDsByEncodedName map[string]string
var ChainIDsByDecodedName map[string]string

var BlockIndexes map[string]string //used to index blocks by both their full and partial hash

type DataStatusStruct struct {
	DBlockHeight int

	//Last DBlock we have seen and saved in an uninterrupted chain
	LastKnownBlock string
	//Last DBlock we have processed and connected back and forth
	LastProcessedBlock string
}

var DataStatus *DataStatusStruct

const DBlocksBucket string = "DBlocks"
const DBlockKeyMRsBySequenceBucket string = "DBlockKeyMRsBySequence"
const BlocksBucket string = "Blocks"
const EntriesBucket string = "Entries"
const ChainsBucket string = "Chains"
const ChainIDsByEncodedNameBucket string = "ChainIDsByEncodedName"
const ChainIDsByDecodedNameBucket string = "ChainIDsByDecodedName"
const BlockIndexesBucket string = "BlockIndexes"
const DataStatusBucket string = "DataStatus"

var BucketList []string = []string{DBlocksBucket, DBlockKeyMRsBySequenceBucket, BlocksBucket, EntriesBucket, ChainsBucket, ChainIDsByEncodedNameBucket, ChainIDsByDecodedNameBucket, BlockIndexesBucket, DataStatusBucket}

func init() {
	DBlocks = map[string]*DBlock{}
	DBlockKeyMRsBySequence = map[string]string{}
	Blocks = map[string]*Block{}
	Entries = map[string]*Entry{}
	BlockIndexes = map[string]string{}
	Chains = map[string]*Chain{}
	ChainIDsByEncodedName = map[string]string{}
	ChainIDsByDecodedName = map[string]string{}

	//DataStatus.LastKnownBlock = "0000000000000000000000000000000000000000000000000000000000000000"
}

type ListEntry struct {
	ChainID string
	KeyMR   string
}

type DBlock struct {
	DBHash string

	PrevBlockKeyMR string
	NextBlockKeyMR string
	Timestamp      uint64
	SequenceNumber int

	EntryBlockList   []ListEntry
	AdminBlock       ListEntry
	FactoidBlock     ListEntry
	EntryCreditBlock ListEntry

	BlockTimeStr string
	KeyMR        string

	Blocks int

	AdminEntries       int
	EntryCreditEntries int
	FactoidEntries     int
	EntryEntries       int
}

func (e *DBlock) JSON() (string, error) {
	return common.EncodeJSONString(e)
}

func (e *DBlock) Spew() string {
	return common.Spew(e)
}

type Common struct {
	ChainID   string
	Timestamp string

	JSONString   string
	SpewString   string
	BinaryString string
}

func (e *Common) JSON() (string, error) {
	return common.EncodeJSONString(e)
}

func (e *Common) Spew() string {
	return common.Spew(e)
}

type Block struct {
	Common

	FullHash    string //KeyMR
	PartialHash string

	PrevBlockHash string
	NextBlockHash string

	EntryCount int

	EntryList []*Entry

	IsAdminBlock       bool
	IsFactoidBlock     bool
	IsEntryCreditBlock bool
	IsEntryBlock       bool
}

func (e *Block) JSON() (string, error) {
	return common.EncodeJSONString(e)
}

func (e *Block) JSONBuffer(b *bytes.Buffer) error {
	return common.EncodeJSONToBuffer(e, b)
}

func (e *Block) Spew() string {
	return common.Spew(e)
}

type Entry struct {
	Common

	ExternalIDs []DecodedString
	Content     DecodedString

	MinuteMarker string

	//Marshallable blocks
	Hash string
}

func (e *Entry) JSON() (string, error) {
	return common.EncodeJSONString(e)
}

func (e *Entry) Spew() string {
	return common.Spew(e)
}

type Chain struct {
	ChainID      string
	Names        []DecodedString
	FirstEntryID string

	//Not saved
	FirstEntry *Entry
}

type DecodedString struct {
	Encoded string
	Decoded string
}

type Address struct {
	Address     string
	AddressType string //EC, Factoid, etc.
	PublicKey   string
	Balance     string
}

//-----------------------------------------------------------------------------------------------
//-------------------------------------Save, load, etc.------------------------------------------
//-----------------------------------------------------------------------------------------------

func RecordChain(block *Block) error {
	if block.PrevBlockHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return nil
	}

	c := new(Chain)
	c.ChainID = block.ChainID
	c.FirstEntryID = block.EntryList[0].Hash
	c.Names = block.EntryList[0].ExternalIDs[:]

	err := SaveChain(c)
	if err != nil {
		return err
	}

	log.Printf("\n\nChain - %v\n\n", c)
	return nil
}

func StoreEntriesFromBlock(block *Block) error {
	for _, v := range block.EntryList {
		err := SaveEntry(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadDBlockKeyMRBySequence(sequence int) (string, error) {
	seq := fmt.Sprintf("%v", sequence)
	keyMR, found := DBlockKeyMRsBySequence[seq]
	if found == true {
		return keyMR, nil
	}

	key := new(string)
	key2, err := LoadData(DBlockKeyMRsBySequenceBucket, seq, key)
	if err != nil {
		return "", err
	}
	if key2 == nil {
		return "", nil
	}
	DBlockKeyMRsBySequence[seq] = *key
	return *key, nil
}

func SaveDBlockKeyMRBySequence(keyMR string, sequence int) error {
	seq := fmt.Sprintf("%v", sequence)
	err := SaveData(DBlockKeyMRsBySequenceBucket, seq, keyMR)
	if err != nil {
		return err
	}
	DBlockKeyMRsBySequence[seq] = keyMR
	return nil
}

//Savers and Loaders
func SaveDBlock(b *DBlock) error {
	err := SaveData(DBlocksBucket, b.KeyMR, b)
	if err != nil {
		return err
	}
	DBlocks[b.KeyMR] = b

	err = SaveDBlockKeyMRBySequence(b.KeyMR, b.SequenceNumber)
	if err != nil {
		return err
	}

	return nil
}

func LoadDBlock(hash string) (*DBlock, error) {
	block, ok := DBlocks[hash]
	if ok == true {
		return block, nil
	}

	block = new(DBlock)
	block2, err := LoadData(DBlocksBucket, hash, block)
	if err != nil {
		return nil, err
	}
	if block2 == nil {
		return nil, nil
	}
	DBlocks[hash] = block
	return block, nil
}

func LoadDBlockBySequence(sequence int) (*DBlock, error) {
	key, err := LoadDBlockKeyMRBySequence(sequence)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}
	return LoadDBlock(key)
}

func SaveBlockIndex(index, hash string) error {
	err := SaveData(BlockIndexesBucket, index, hash)
	if err != nil {
		return err
	}
	BlockIndexes[index] = hash
	return nil
}

func LoadBlockIndex(hash string) (string, error) {
	index, found := BlockIndexes[hash]
	if found == true {
		return index, nil
	}

	ind := new(string)
	ind2, err := LoadData(BlockIndexesBucket, hash, ind)
	if err != nil {
		return "", err
	}
	if ind2 == nil {
		return "", nil
	}
	BlockIndexes[hash] = *ind
	return *ind, nil
}

func SaveBlock(b *Block) error {
	StoreEntriesFromBlock(b)

	err := SaveBlockIndex(b.FullHash, b.PartialHash)
	if err != nil {
		return err
	}
	err = SaveBlockIndex(b.PartialHash, b.PartialHash)
	if err != nil {
		return err
	}

	err = SaveData(BlocksBucket, b.PartialHash, b)
	if err != nil {
		return err
	}
	Blocks[b.PartialHash] = b

	if b.IsEntryBlock {
		RecordChain(b)
	}

	return nil
}

func LoadBlock(hash string) (*Block, error) {
	key, err := LoadBlockIndex(hash)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}

	block, ok := Blocks[key]
	if ok == true {
		return block, nil
	}

	block = new(Block)
	block2, err := LoadData(BlocksBucket, key, block)
	if err != nil {
		return nil, err
	}
	if block2 == nil {
		return nil, nil
	}
	Blocks[key] = block
	Blocks[hash] = block
	return block, nil
}

func SaveEntry(e *Entry) error {
	err := SaveData(EntriesBucket, e.Hash, e)
	if err != nil {
		return err
	}
	Entries[e.Hash] = e
	return nil
}

func LoadEntry(hash string) (*Entry, error) {
	entry, found := Entries[hash]
	if found == true {
		return entry, nil
	}

	entry = new(Entry)
	entry2, err := LoadData(EntriesBucket, hash, entry)
	if err != nil {
		return nil, err
	}
	if entry2 == nil {
		return nil, nil
	}
	Entries[hash] = entry
	return entry, nil
}

func SaveChainIDsByName(chainID, decodedName, encodedName string) error {
	err := SaveData(ChainIDsByDecodedNameBucket, decodedName, chainID)
	if err != nil {
		return err
	}
	ChainIDsByDecodedName[decodedName] = chainID
	err = SaveData(ChainIDsByEncodedNameBucket, encodedName, chainID)
	if err != nil {
		return err
	}
	ChainIDsByEncodedName[encodedName] = chainID
	return nil
}

func LoadChainIDByName(name string) (string, error) {
	id, found := ChainIDsByDecodedName[name]
	if found == true {
		return id, nil
	}

	entry := new(string)
	entry2, err := LoadData(ChainIDsByDecodedNameBucket, name, entry)
	if err != nil {
		return "", err
	}
	if entry2 != nil {
		ChainIDsByDecodedName[name] = *entry
		return *entry, nil
	}

	id, found = ChainIDsByEncodedName[name]
	if found == true {
		return id, nil
	}

	entry = new(string)
	entry2, err = LoadData(ChainIDsByEncodedNameBucket, name, entry)
	if err != nil {
		return "", err
	}
	if entry2 != nil {
		ChainIDsByEncodedName[name] = *entry
		return *entry, nil
	}

	return "", nil
}

func SaveChain(c *Chain) error {
	err := SaveData(ChainsBucket, c.ChainID, c)
	if err != nil {
		return err
	}
	Chains[c.ChainID] = c

	for _, v := range c.Names {
		err = SaveChainIDsByName(c.ChainID, v.Decoded, v.Encoded)
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadChain(hash string) (*Chain, error) {
	chain, found := Chains[hash]
	if found == true {
		return chain, nil
	}

	chain = new(Chain)
	var err error
	_, err = LoadData(ChainsBucket, hash, chain)
	if err != nil {
		return nil, err
	}
	Chains[hash] = chain
	return chain, nil
}

func SaveDataStatus(ds *DataStatusStruct) error {
	err := SaveData(DataStatusBucket, DataStatusBucket, ds)
	if err != nil {
		return err
	}
	DataStatus = ds
	return nil
}

func LoadDataStatus() *DataStatusStruct {
	if DataStatus != nil {
		return DataStatus
	}
	ds := new(DataStatusStruct)
	var err error
	ds2, err := LoadData(DataStatusBucket, DataStatusBucket, ds)
	if err != nil {
		panic(err)
	}
	if ds2 == nil {
		ds = new(DataStatusStruct)
		ds.LastKnownBlock = "0000000000000000000000000000000000000000000000000000000000000000"
		ds.LastProcessedBlock = "0000000000000000000000000000000000000000000000000000000000000000"
	}
	DataStatus = ds
	log.Printf("LoadDataStatus DS - %v, %v", ds, ds2)
	return ds
}

//Getters

func GetBlock(hash string) (*Block, error) {
	hash = strings.ToLower(hash)

	block, err := LoadBlock(hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("Block %v not found", hash)
	}
	return block, nil
}

func GetBlockHeight() int {
	return LoadDataStatus().DBlockHeight
}

func GetDBlocks(start, max int) ([]*DBlock, error) {
	answer := []*DBlock{}
	for i := start; i <= max; i++ {
		block, err := LoadDBlockBySequence(i)
		if err != nil {
			return nil, err
		}
		if block == nil {
			continue
		}
		answer = append(answer, block)
	}
	return answer, nil
}

func GetDBlock(keyMR string) (*DBlock, error) {
	keyMR = strings.ToLower(keyMR)

	block, err := LoadDBlock(keyMR)
	if err != nil {
		return nil, err
	}
	return block, nil
}

type DBInfo struct {
	BTCTxHash string
}

//Getters

func GetEntry(hash string) (*Entry, error) {
	hash = strings.ToLower(hash)
	entry, err := LoadEntry(hash)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		//str, _ := EncodeJSONString(Entries)
		//log.Printf("%v not found in %v", hash, str)
		return nil, errors.New("Entry not found")
	}
	return entry, nil
}

func GetDBInfo(keyMR string) (DBInfo, error) {
	//TODO: gather DBInfo
	return DBInfo{}, nil
}

func GetChains() ([]*Chain, error) {
	//TODO: load chains from database
	answer := []*Chain{}
	for _, v := range Chains {
		answer = append(answer, v)
	}
	return answer, nil
}

func GetChain(hash string) (*Chain, error) {
	hash = strings.ToLower(hash)
	chain, err := LoadChain(hash)
	if err != nil {
		return nil, err
	}
	if chain == nil {
		return chain, errors.New("Chain not found")
	}
	entry, err := LoadEntry(chain.FirstEntryID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return chain, errors.New("First entry not found")
	}
	chain.FirstEntry = entry
	return chain, nil
}

func GetChainByName(name string) (*Chain, error) {
	id, err := LoadChainIDByName(name)
	if err != nil {
		return nil, err
	}
	if id != "" {
		return GetChain(id)
	}

	return GetChain(name)
}

type EBlock struct {
	factom.EBlock
}
