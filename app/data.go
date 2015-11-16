package main

import (
	//"appengine"
	//"appengine/datastore"
	"bytes"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
	"github.com/couchbase/gocb"
	"github.com/mitchellh/mapstructure"
	"reflect"
	//"log"
	//"github.com/ThePiachu/Go/Datastore"
	//"github.com/ThePiachu/Go/Log"
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
	//Last DBlock we tallied balances in
	LastTalliedBlockNumber int
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

type ListEntry struct {
	ChainID string
	KeyMR   string
}

type DBlock struct {
	DBHash string

	PrevBlockKeyMR string
	NextBlockKeyMR string
	Timestamp      int64
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

	FactoidTally string
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

	EntryIDList []string
	EntryList   []*Entry

	IsAdminBlock       bool
	IsFactoidBlock     bool
	IsEntryCreditBlock bool
	IsEntryBlock       bool

	TotalIns   string
	TotalOuts  string
	TotalECs   string
	TotalDelta string

	Created   string
	Destroyed string
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
	Content     DecodedString

	MinuteMarker string

	//Marshallable blocks
	Hash string

	//Anchor chain-specific data
	AnchorRecord *AnchorRecord

	TotalIns  string
	TotalOuts string
	TotalECs  string

	Delta string
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
	Entries    []*Entry
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


func RecordChain(block *Block) error {
	fmt.Errorf("RecordChain")
	if block.PrevBlockHash != ZeroID {
		fmt.Errorf("block.PrevBlockHash != ZeroID")
		return nil
	}

	chain := new(Chain)
	chain.ChainID = block.ChainID
	chain.FirstEntryID = block.EntryList[0].Hash
	chain.Names = block.EntryList[0].ExternalIDs[:]

	err := SaveChain(chain)
	if err != nil {
		fmt.Errorf("StoreEntriesFromBlock - %v", err)
		return err
	}

	fmt.Errorf("Chain - %v", chain)
	return nil
}

func StoreEntriesFromBlock(block *Block) error {
	block.EntryIDList = make([]string, len(block.EntryList))
	for i, v := range block.EntryList {
		err := SaveEntry(v)
		if err != nil {
			fmt.Errorf("StoreEntriesFromBlock - %v", err)
			return err
		}
		block.EntryIDList[i] = v.Hash
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
    var mapResults map[string]interface{}
    blockIdx := new(BlockIndex)
	
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + seq + "\" AND DataType=\"" + DBlockKeyMRsBySequenceBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }
    var row interface{}
    for rows.Next(&row) {
        if row != nil {
            mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
            err := mapstructure.Decode(mapResults, &blockIdx)
            if err != nil {
                panic(err)
            }
            *newKey = blockIdx.BlockIndex
        }
        
    }
    rows.Close()

	DBlockKeyMRsBySequence[seq] = *newKey

    
	return *newKey, nil
}



func SaveDBlockKeyMRBySequence(keyMR string, sequence int) error {
	seq := fmt.Sprintf("%v", sequence)
	err := SaveData(DBlockKeyMRsBySequenceBucket, seq, &BlockIndex{BlockIndex: keyMR})
	if err != nil {
		fmt.Errorf("SaveDBlockKeyMRBySequence - %v", err)
		return err
	}
	DBlockKeyMRsBySequence[seq] = keyMR

	return nil
}

//Savers and Loaders
func SaveDBlock(b *DBlock) error {
	err := SaveData(DBlocksBucket, b.KeyMR, b)
	if err != nil {
		fmt.Errorf("SaveDBlock - %v", err)
		return err
	}

	DBlocks[b.KeyMR] = b

	err = SaveDBlockKeyMRBySequence(b.KeyMR, b.SequenceNumber)
	if err != nil {
		fmt.Errorf("SaveDBlock - %v", err)
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

type BlockIndex struct {
	BlockIndex string
}

func SaveBlockIndex(index, hash string) error {

	err := SaveData(BlockIndexesBucket, index, &BlockIndex{BlockIndex: hash})
	if err != nil {
		fmt.Errorf("SaveBlockIndex - %v", err)
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

    blockIdx := new(BlockIndex)
    strBlockIdx := new(string)
    var row interface{}
    var tempRow interface{}
	var mapResults map[string]interface{}
	query := gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE META(default).id = \"" + hash + "\" AND DataType=\"" + BlockIndexesBucket + "\";")
    rows, qryErr := myBucket.ExecuteN1qlQuery(query, nil)
    if qryErr != nil {
        fmt.Printf("QUERY ERROR: ", qryErr)
    }    
    if !rows.Next(&tempRow) {
        query = gocb.NewN1qlQuery("SELECT DataContent FROM `default` WHERE DataContent.BlockIndex = \"" + hash + "\" AND DataType=\"" + BlockIndexesBucket + "\";")
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

    if row != nil {
        mapResults = row.(map[string]interface{})["DataContent"].(map[string]interface{})
        err := mapstructure.Decode(mapResults, &blockIdx)
        if err != nil {
            panic(err)
        }
        *strBlockIdx = blockIdx.BlockIndex
    }

    rows.Close()
	return *strBlockIdx, nil
}

func SaveBlock(b *Block) error {
	StoreEntriesFromBlock(b)

	err := SaveBlockIndex(b.FullHash, b.PartialHash)
	if err != nil {
		fmt.Errorf("SaveBlock - %v", err)
		return err
	}
	err = SaveBlockIndex(b.PartialHash, b.PartialHash)
	if err != nil {
		fmt.Errorf("SaveBlock - %v", err)
		return err
	}

	err = SaveData(BlocksBucket, b.PartialHash, b)
	if err != nil {
		fmt.Errorf("SaveBlock - %v", err)
		return err
	}

	Blocks[b.PartialHash] = b


	if b.IsEntryBlock {
		err = RecordChain(b)
		if err != nil {
			return err
		}
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

    //THE FOLLOWING 3 LINES (& FUNCTION) ARE FROM PIACHU
	err = LoadBlockEntries(newBlock)
	if err != nil {
		fmt.Errorf("LoadBlock - %v", err)
	}
	

	Blocks[key] = newBlock
	Blocks[hash] = newBlock
    
	return newBlock, nil
}


func LoadBlockEntries(block *Block) error {
	if len(block.EntryList) > 0 {
		return nil
	}
	entries := make([]*Entry, len(block.EntryIDList))
	for i, v := range block.EntryIDList {
		entry, err := LoadEntry(v)
		if err != nil {
			return err
		}
		entries[i] = entry
	}
	block.EntryList = entries
	return nil
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

    //THE FOLLOWING 8 LINES ARE FROM PIACHU
	if newEntry.ChainID == AnchorBlockID {
		ar, err := ParseAnchorChainData(newEntry.Content.Decoded)
		if err != nil {
			fmt.Errorf("LoadEntry - %v", err)
			return nil, err
		}
		entry.AnchorRecord = ar
	}
	
	Entries[hash] = newEntry
	
	return newEntry, nil
}

func ParseAnchorChainData(data string) (*AnchorRecord, error) {
	if len(data) < 128 {
		return nil, nil
	}
	tmp := data[:len(data)-128]
	ar := new(AnchorRecord)

	err := common.DecodeJSONString(tmp, ar)
	if err != nil {
		fmt.Println("ParseAnchorChainData - %v", err)
		return nil, err
	}
	return ar, nil
}

func SaveChainIDsByName(chainID, decodedName, encodedName string) error {
	err := SaveData(ChainIDsByDecodedNameBucket, decodedName, &BlockIndex{BlockIndex: chainID})
	if err != nil {
		fmt.Errorf("SaveChainIDsByName - %v", err)
		return err
	}
	
	ChainIDsByDecodedName[decodedName] = chainID


	err = SaveData(ChainIDsByEncodedNameBucket, encodedName, &BlockIndex{BlockIndex: chainID})
	if err != nil {
		fmt.Errorf("SaveChainIDsByName - %v", err)
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


func SaveChain(chain *Chain) error {
	err := SaveData(ChainsBucket, chain.ChainID, chain)
	if err != nil {
		fmt.Errorf("SaveChain - %v", err)
		return err
	}
	
	Chains[chain.ChainID] = chain

	for _, v := range chain.Names {
		err = SaveChainIDsByName(chain.ChainID, v.Decoded, v.Encoded)
		if err != nil {
			fmt.Errorf("SaveChain - %v", err)
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
	
	DataStatus = couchDS
	fmt.Printf("LoadDataStatus DS - %v\n", couchDS)
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
		//fmt.Errorf("%v not found in %v", hash, str)
		return nil, errors.New("Entry not found")
	}
	return entry, nil
}

func GetChains() ([]Chain, error) {
	//TODO: load chains from database

	answer := []Chain{}
	
	//TEMPORARY BUTCHER
	//_, err := Datastore.QueryGetAll(ChainsBucket, &answer)
	err := fmt.Errorf("Error is not nil - BUTCHER")
	if err != nil {
		fmt.Errorf("GetChains - %v", err)
		return nil, err
	}
	return answer, nil
}

func GetChain(hash string, startFrom, amountToFetch int) (*Chain, error) {
	fmt.Errorf("GetChain - %v, %v, %v", hash, startFrom, amountToFetch)
	hash = strings.ToLower(hash)
	chain, err := LoadChain(hash)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		return nil, err
	}
	if chain == nil {
		return nil, errors.New("Chain not found")
	}
	entry, err := LoadEntry(chain.FirstEntryID)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		return nil, err
	}
	if entry == nil {
		fmt.Errorf("Error - %v", err)
		return nil, errors.New("First entry not found")
	}
	chain.FirstEntry = entry

	entries, err := GetChainEntries(chain.ChainID, startFrom, amountToFetch)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		return nil, err
	}
	chain.Entries = entries

	return chain, nil
}

func GetChainByName(name string, startFrom, amountToFetch int) (*Chain, error) {
	fmt.Errorf("GetChainByName - %v, %v, %v", name, startFrom, amountToFetch)
	id, err := LoadChainIDByName(name)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		return nil, err
	}
	if id != "" {
		return GetChain(id, startFrom, amountToFetch)
	}

	return GetChain(name, startFrom, amountToFetch)
}

type EBlock struct {
	factom.EBlock
}

func GetChainEntries(chainID string, startFrom, amountToFetch int) ([]*Entry, error) {
	//tmp := []Entry{}
	//TEMPORARY BUTCHER
	//keys, err := Datastore.QueryGetAllKeysWithFilterLimitOffsetAndOrder(EntriesBucket, "ChainID=", chainID, amountToFetch, startFrom, "Timestamp", &tmp)
    keys := []string{"a", "b"}
	err := fmt.Errorf("Error is not nil - BUTCHER")
	if err != nil {
		return nil, err
	}
	answer := make([]*Entry, len(keys))
	for i, v := range keys {
	    //TEMPORARY BUTCHER
		//entry, err := GetEntry(v.StringID())
        entry, err := GetEntry(v)
		if err != nil {
			return nil, err
		}
		answer[i] = entry
	}
	return answer, nil
}

func ListExternalIDs() (map[string][]string, error) {
	//tmp := []Entry{}
	//TEMPORARY BUTCHER
	//keys := Datastore.QueryGetAllKeysWithFilter(EntriesBucket, "ExternalIDs.Encoded>", "", tmp)
    keys := []string{"a", "b"}
    answer := map[string][]string{}
	for _, v := range keys {
	    //TEMPORARY BUTCHER
		//entry, err := LoadEntry(v.StringID())
		entry, err := LoadEntry(v)
		if err != nil {
			fmt.Errorf("ListExternalIDs - %v", err)
			return nil, err
		}
		for _, exID := range entry.ExternalIDs {
			list, ok := answer[exID.Decoded]
			if ok == false {
				list = []string{}
			}
			found := false
			for i := range list {
				if list[i] == entry.Hash {
					found = true
					break
				}
			}
			if found == false {
				list = append(list, entry.Hash)
				answer[exID.Decoded] = list
			}

			list, ok = answer[exID.Encoded]
			if ok == false {
				list = []string{}
			}
			found = false
			for i := range list {
				if list[i] == entry.Hash {
					found = true
					break
				}
			}
			if found == false {
				list = append(list, entry.Hash)
				answer[exID.Encoded] = list
			}
		}
	}

	return answer, nil
}
