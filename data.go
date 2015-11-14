package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
	"github.com/couchbase/gocb"
	"github.com/mitchellh/mapstructure"
	"reflect"
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

	AnchoredInTransaction string
	AnchorRecord          string

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

	ShortEntry string //a way to replace the entry with a short string

	ExternalIDs []DecodedString
	Content     *DecodedString

	MinuteMarker string

	//Marshallable blocks
	Hash string

	//Anchor chain-specific data
	AnchorRecord *AnchorRecord
}

type AnchorRecord struct {
	AnchorRecordVer int
	DBHeight        uint32
	KeyMR           string
	RecordHeight    uint32

	Bitcoin struct {
		Address     string //"1HLoD9E4SDFFPDiYfNYnkBLQ85Y51J3Zb1",
		TXID        string //"9b0fc92260312ce44e74ef369f5c66bbb85848f2eddd5a7a1cde251e54ccfdd5", BTC Hash - in reverse byte order
		BlockHeight int32  //345678,
		BlockHash   string //"00000000000000000cc14eacfc7057300aea87bed6fee904fd8e1c1f3dc008d4", BTC Hash - in reverse byte order
		Offset      int32  //87
	}
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

	newKey := new(string)
	
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + seq + "\" AND DataType=\"" + DBlockKeyMRsBySequenceBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {

        
        *newKey = row.(map[string]interface{})["DataContent"].(string)
        
    }
    rows.Close()
    
	DBlockKeyMRsBySequence[seq] = *newKey
	return *newKey, nil
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

	newBlock := new(DBlock)
    var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + hash + "\" AND DataType=\"" + DBlocksBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        
        err := mapstructure.Decode(mapResults, &newBlock)
        if err != nil {
            panic(err)
        }
        
    }
    rows.Close()

    if reflect.DeepEqual(newBlock, new(DBlock)) {
        //newBlock is empty (zero'd)
    	return nil, nil
    }
	
	DBlocks[hash] = newBlock
	return newBlock, nil
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

    blockIdx := new(string)
    var row interface{}
    var tempRow interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + hash + "\" AND DataType=\"" + BlockIndexesBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }    
    if !rows.Next(&tempRow) {
        query = gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE DataContent = \"" + hash + "\" AND DataType=\"" + BlockIndexesBucket + "\";")
        rows, qryErr = myBucket.ExecuteN1qlQuery(query, nil)
        if qryErr != nil {
            fmt.Printf("QUERY ERROR: ", qryErr)
        }
        if !rows.Next(&row) {
            fmt.Printf("BlockIndexesBucket doesn't contain any references to ", hash)
        }
    } else {
        row = tempRow
    }

    *blockIdx = row.(map[string]interface{})["DataContent"].(string)

    rows.Close()

    BlockIndexes[hash] = *blockIdx
	return *blockIdx, nil
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

	newBlock := new(Block)
	newCommon := new(Common)

	var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + key + "\" AND DataType=\"" + BlocksBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        err = mapstructure.Decode(mapResults, &newCommon)
        if err != nil {
            panic(err)
        }
        
        err = mapstructure.Decode(mapResults, &newBlock)
        if err != nil {
            panic(err)
        }
        
        newBlock.Common = *newCommon
    }
    rows.Close()
    
	Blocks[key] = newBlock
	Blocks[hash] = newBlock
	return newBlock, nil
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

	newEntry := new(Entry)
	newCommon := new(Common)

	var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + hash + "\" AND DataType=\"" + EntriesBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        err := mapstructure.Decode(mapResults, &newCommon)
        if err != nil {
            panic(err)
        }
        
        err = mapstructure.Decode(mapResults, &newEntry)
        if err != nil {
            panic(err)
        }
        
        newEntry.Common = *newCommon
    }
    rows.Close()
	
	Entries[hash] = newEntry
	return newEntry, nil
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

	newEntry := new(string)
	
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + name + "\" AND DataType=\"" + ChainIDsByDecodedNameBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        *newEntry = row.(map[string]interface{})["DataContent"].(string)
        
    }
    rows.Close()
    
	if len(*newEntry) > 0 {
	    ChainIDsByDecodedName[name] = *newEntry
	    return *newEntry, nil
	}
	
	id, found = ChainIDsByEncodedName[name]
	if found == true {
		return id, nil
	}
	
	query = gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + name + "\" AND DataType=\"" + ChainIDsByEncodedNameBucket + "\";")
    rows, qryErr = myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    for rows.Next(&row) {
        *newEntry = row.(map[string]interface{})["DataContent"].(string)
        
    }
    rows.Close()
    
	if len(*newEntry) > 0 {
	    ChainIDsByEncodedName[name] = *newEntry
	    return *newEntry, nil
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

	newChain := new(Chain)
	
    var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + hash + "\" AND DataType=\"" + ChainsBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        
        err := mapstructure.Decode(mapResults, &newChain)
        if err != nil {
            panic(err)
        }
        
    }
    rows.Close()
	
	Chains[hash] = newChain
	return newChain, nil
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
	var err error
	couchDS := new(DataStatusStruct)
    var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"DataStatus\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        
        err = mapstructure.Decode(mapResults, &couchDS)
        if err != nil {
            panic(err)
        }
    }
    rows.Close()

	if couchDS == nil {
		couchDS = new(DataStatusStruct)
		couchDS.LastKnownBlock = "0000000000000000000000000000000000000000000000000000000000000000"
		couchDS.LastProcessedBlock = "0000000000000000000000000000000000000000000000000000000000000000"
	}
	
	return couchDS
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

func GetDBlocksReverseOrder(start, max int) ([]*DBlock, error) {
	blocks, err := GetDBlocks(start, max)
	if err != nil {
		return nil, err
	}
	answer := make([]*DBlock, len(blocks))
	for i := range blocks {
		answer[len(blocks)-1-i] = blocks[i]
	}
	return answer, nil
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
