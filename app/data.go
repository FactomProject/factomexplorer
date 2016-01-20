package app

import (
	"appengine"
	"appengine/datastore"
	"bytes"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
	"github.com/ThePiachu/Go/Datastore"
	"github.com/ThePiachu/Go/Log"
	"strings"
)

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

func NukeDatabase(c appengine.Context) error {
	toDelete := []string{
		DBlocksBucket,
		DBlockKeyMRsBySequenceBucket,
		BlocksBucket,
		EntriesBucket,
		ChainsBucket,
		ChainIDsByEncodedNameBucket,
		ChainIDsByDecodedNameBucket,
		BlockIndexesBucket,
		DataStatusBucket,
	}
	for _, v := range toDelete {
		err := Datastore.ClearNamespace(c, v)
		if err != nil {
			return err
		}
	}
	err := Datastore.FlushMemcache(c)
	if err != nil {
		return err
	}
	return nil
}

var BucketList []string = []string{DBlocksBucket, DBlockKeyMRsBySequenceBucket, BlocksBucket, EntriesBucket, ChainsBucket, ChainIDsByEncodedNameBucket, ChainIDsByDecodedNameBucket, BlockIndexesBucket, DataStatusBucket}

type ListEntry struct {
	ChainID string
	KeyMR   string
}

type DBlock struct {
	DBHash string

	PrevBlockKeyMR string `datastore:",noindex"`
	NextBlockKeyMR string `datastore:",noindex"`
	Timestamp      int64
	SequenceNumber int

	EntryBlockList   []ListEntry `datastore:",noindex"`
	AdminBlock       ListEntry   `datastore:",noindex"`
	FactoidBlock     ListEntry   `datastore:",noindex"`
	EntryCreditBlock ListEntry   `datastore:",noindex"`

	BlockTimeStr string
	KeyMR        string

	AnchoredInTransaction string `datastore:",noindex"`
	AnchorRecord          string `datastore:",noindex"`

	Blocks int

	AdminEntries       int `datastore:",noindex"`
	EntryCreditEntries int `datastore:",noindex"`
	FactoidEntries     int `datastore:",noindex"`
	EntryEntries       int `datastore:",noindex"`

	FactoidTally string `datastore:",noindex"`
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

	JSONString   string `datastore:",noindex"`
	BinaryString string `datastore:",noindex"`
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

	PrevBlockHash string `datastore:",noindex"`
	NextBlockHash string `datastore:",noindex"`

	EntryCount int `datastore:",noindex"`

	EntryIDList []string `datastore:",noindex"`
	EntryList   []*Entry `datastore:"-"`

	IsAdminBlock       bool `datastore:",noindex"`
	IsFactoidBlock     bool `datastore:",noindex"`
	IsEntryCreditBlock bool `datastore:",noindex"`
	IsEntryBlock       bool `datastore:",noindex"`

	TotalIns   string `datastore:",noindex"`
	TotalOuts  string `datastore:",noindex"`
	TotalECs   string `datastore:",noindex"`
	TotalDelta string `datastore:",noindex"`

	Created   string `datastore:",noindex"`
	Destroyed string `datastore:",noindex"`
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
	Content     DecodedString `datastore:",noindex"`

	MinuteMarker string `datastore:",noindex"`

	//Marshallable blocks
	Hash string

	//Anchor chain-specific data
	AnchorRecord *AnchorRecord `datastore:"-"`

	TotalIns  string `datastore:",noindex"`
	TotalOuts string `datastore:",noindex"`
	TotalECs  string `datastore:",noindex"`

	Delta string `datastore:",noindex"`
}

func (e *Entry) LoadStrings() {
	for i:=range(e.ExternalIDs) {
		e.ExternalIDs[i].LoadStrings()
	}
}

func (e *Entry) Trim() {
	for i:=range(e.ExternalIDs) {
		e.ExternalIDs[i].Trim()
	}
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
	FirstEntryID string `datastore:",noindex"`

	//Not saved
	FirstEntry *Entry   `datastore:"-"`
	Entries    []*Entry `datastore:"-"`
}

func (e *Chain) LoadStrings() {
	for i:=range(e.Names) {
		e.Names[i].LoadStrings()
	}
}

func (e *Chain) Trim() {
	for i:=range(e.Names) {
		e.Names[i].Trim()
	}
}

type DecodedString struct {
	Encoded    string
	Decoded    string
	NonIndexed []byte `datastore:",noindex"`
}

func (ds *DecodedString) LoadStrings() {
	if len(ds.NonIndexed) > 0 {
		ds.Encoded = fmt.Sprintf("%x", ds.NonIndexed)
		ds.Decoded = string(ds.NonIndexed)
		if appengine.IsDevAppServer() {
			ds.Decoded = SanitizeKey(ds.Decoded)
		}
	}
}

func (ds *DecodedString) Trim() {
	max:=1500
	if len(ds.Encoded) > max {
		ds.NonIndexed = []byte(ds.Decoded)
		ds.Encoded = ds.Encoded[:max]
	}
	if len(ds.Decoded) > max {
		ds.NonIndexed = []byte(ds.Decoded)
		ds.Decoded = ds.Decoded[:max]
	}
}

func (ds *DecodedString) Save(c chan<- datastore.Property) error {
	defer close(c)
	ds.Trim()
	return datastore.SaveStruct(ds, c)
}

func (ds *DecodedString) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(ds, c); err != nil {
		return err
	}
	ds.LoadStrings()
	return nil
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

func RecordChain(c appengine.Context, block *Block) error {
	Log.Debugf(c, "RecordChain")
	if block.PrevBlockHash != ZeroID {
		Log.Debugf(c, "block.PrevBlockHash != ZeroID")
		return nil
	}

	chain := new(Chain)
	chain.ChainID = block.ChainID
	chain.FirstEntryID = block.EntryList[0].Hash
	chain.Names = block.EntryList[0].ExternalIDs[:]

	err := SaveChain(c, chain)
	if err != nil {
		Log.Errorf(c, "StoreEntriesFromBlock - %v", err)
		return err
	}

	Log.Debugf(c, "Chain - %v", chain)
	return nil
}

func StoreEntriesFromBlock(c appengine.Context, block *Block) error {
	block.EntryIDList = make([]string, len(block.EntryList))
	for i, v := range block.EntryList {
		err := SaveEntry(c, v)
		if err != nil {
			Log.Errorf(c, "StoreEntriesFromBlock - %v", err)
			return err
		}
		block.EntryIDList[i] = v.Hash
	}
	return nil
}

func LoadDBlockKeyMRBySequence(c appengine.Context, sequence int) (string, error) {
	seq := fmt.Sprintf("%v", sequence)

	key := new(Index)
	key2, err := LoadData(c, DBlockKeyMRsBySequenceBucket, seq, key)
	if err != nil {
		return "", err
	}
	if key2 == nil {
		return "", nil
	}
	return key.Index, nil
}

func SaveDBlockKeyMRBySequence(c appengine.Context, keyMR string, sequence int) error {
	seq := fmt.Sprintf("%v", sequence)
	err := SaveData(c, DBlockKeyMRsBySequenceBucket, seq, &Index{Index: keyMR})
	if err != nil {
		Log.Errorf(c, "SaveDBlockKeyMRBySequence - %v", err)
		return err
	}
	return nil
}

//Savers and Loaders
func SaveDBlock(c appengine.Context, b *DBlock) error {
	err := SaveData(c, DBlocksBucket, b.KeyMR, b)
	if err != nil {
		Log.Errorf(c, "SaveDBlock - %v", err)
		return err
	}

	err = SaveDBlockKeyMRBySequence(c, b.KeyMR, b.SequenceNumber)
	if err != nil {
		Log.Errorf(c, "SaveDBlock - %v", err)
		return err
	}

	return nil
}

func LoadDBlock(c appengine.Context, hash string) (*DBlock, error) {
	block := new(DBlock)
	block2, err := LoadData(c, DBlocksBucket, hash, block)
	if err != nil {
		return nil, err
	}
	if block2 == nil {
		return nil, nil
	}

	return block, nil
}

func LoadDBlockBySequence(c appengine.Context, sequence int) (*DBlock, error) {
	key, err := LoadDBlockKeyMRBySequence(c, sequence)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}
	return LoadDBlock(c, key)
}

type Index struct {
	Index string
}

func SaveBlockIndex(c appengine.Context, index, hash string) error {

	err := SaveData(c, BlockIndexesBucket, index, &Index{Index: hash})
	if err != nil {
		Log.Errorf(c, "SaveBlockIndex - %v", err)
		return err
	}

	return nil
}

func LoadBlockIndex(c appengine.Context, hash string) (string, error) {
	ind := new(Index)
	ind2, err := LoadData(c, BlockIndexesBucket, hash, ind)
	if err != nil {
		return "", err
	}
	if ind2 == nil {
		return "", nil
	}

	return ind.Index, nil
}

func SaveBlock(c appengine.Context, b *Block) error {
	StoreEntriesFromBlock(c, b)

	err := SaveBlockIndex(c, b.FullHash, b.PartialHash)
	if err != nil {
		Log.Errorf(c, "SaveBlock - %v", err)
		return err
	}
	err = SaveBlockIndex(c, b.PartialHash, b.PartialHash)
	if err != nil {
		Log.Errorf(c, "SaveBlock - %v", err)
		return err
	}

	err = SaveData(c, BlocksBucket, b.PartialHash, b)
	if err != nil {
		Log.Errorf(c, "SaveBlock - %v", err)
		return err
	}

	if b.IsEntryBlock {
		err = RecordChain(c, b)
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadBlock(c appengine.Context, hash string) (*Block, error) {
	key, err := LoadBlockIndex(c, hash)
	if err != nil {
		Log.Errorf(c, "LoadBlock - %v", err)
		return nil, err
	}
	if key == "" {
		return nil, nil
	}

	block := new(Block)
	block2, err := LoadData(c, BlocksBucket, key, block)
	if err != nil {
		Log.Errorf(c, "LoadBlock - %v", err)
		return nil, err
	}
	if block2 == nil {
		return nil, nil
	}
	err = LoadBlockEntries(c, block)
	if err != nil {
		Log.Errorf(c, "LoadBlock - %v", err)
	}

	return block, nil
}

func LoadBlockEntries(c appengine.Context, block *Block) error {
	if len(block.EntryList) > 0 {
		return nil
	}
	entries := make([]*Entry, len(block.EntryIDList))
	for i, v := range block.EntryIDList {
		entry, err := LoadEntry(c, v)
		if err != nil {
			return err
		}
		entries[i] = entry
	}
	block.EntryList = entries
	return nil
}

func SaveEntry(c appengine.Context, e *Entry) error {
	e.Trim()
	err := SaveData(c, EntriesBucket, e.Hash, e)
	if err != nil {
		return err
	}

	return nil
}

func LoadEntry(c appengine.Context, hash string) (*Entry, error) {
	entry := new(Entry)
	entry2, err := LoadData(c, EntriesBucket, hash, entry)
	if err != nil {
		return nil, err
	}
	if entry2 == nil {
		return nil, nil
	}
	if entry.ChainID == AnchorBlockID {
		ar, err := ParseAnchorChainData(c, entry.Content.Decoded)
		if err != nil {
			Log.Errorf(c, "LoadEntry - %v", err)
			return nil, err
		}
		entry.AnchorRecord = ar
	}
	entry.LoadStrings()

	return entry, nil
}

func ParseAnchorChainData(c appengine.Context, data string) (*AnchorRecord, error) {
	if len(data) < 128 {
		return nil, nil
	}
	tmp := data[:len(data)-128]
	ar := new(AnchorRecord)

	err := common.DecodeJSONString(tmp, ar)
	if err != nil {
		Log.Infof(c, "ParseAnchorChainData - %v", err)
		return nil, err
	}
	return ar, nil
}

func SaveChainIDsByName(c appengine.Context, chainID, decodedName, encodedName string) error {
	err := SaveData(c, ChainIDsByDecodedNameBucket, decodedName, &Index{Index: chainID})
	if err != nil {
		Log.Errorf(c, "SaveChainIDsByName - %v", err)
		return err
	}

	err = SaveData(c, ChainIDsByEncodedNameBucket, encodedName, &Index{Index: chainID})
	if err != nil {
		Log.Errorf(c, "SaveChainIDsByName - %v", err)
		return err
	}

	return nil
}

func LoadChainIDByName(c appengine.Context, name string) (string, error) {
	entry := new(Index)
	entry2, err := LoadData(c, ChainIDsByDecodedNameBucket, name, entry)
	if err != nil {
		Log.Errorf(c, "LoadChainIDByName - %v", err)
		return "", err
	}
	if entry2 != nil {
		return entry.Index, nil
	}

	entry = new(Index)
	entry2, err = LoadData(c, ChainIDsByEncodedNameBucket, name, entry)
	if err != nil {
		Log.Errorf(c, "LoadChainIDByName - %v", err)
		return "", err
	}
	if entry2 != nil {
		return entry.Index, nil
	}

	return "", nil
}

func SaveChain(c appengine.Context, chain *Chain) error {
	chain.Trim()
	err := SaveData(c, ChainsBucket, chain.ChainID, chain)
	if err != nil {
		Log.Errorf(c, "SaveChain - %v", err)
		return err
	}

	for _, v := range chain.Names {
		err = SaveChainIDsByName(c, chain.ChainID, v.Decoded, v.Encoded)
		if err != nil {
			Log.Errorf(c, "SaveChain - %v", err)
			return err
		}
	}

	return nil
}

func LoadChain(c appengine.Context, hash string) (*Chain, error) {
	chain := new(Chain)
	var err error
	_, err = LoadData(c, ChainsBucket, hash, chain)
	if err != nil {
		return nil, err
	}
	chain.LoadStrings()

	return chain, nil
}

func SaveDataStatus(c appengine.Context, ds *DataStatusStruct) error {
	err := SaveData(c, DataStatusBucket, DataStatusBucket, ds)
	if err != nil {
		return err
	}
	DataStatus = ds
	return nil
}

func LoadDataStatus(c appengine.Context) *DataStatusStruct {
	if DataStatus != nil {
		return DataStatus
	}
	ds := new(DataStatusStruct)
	var err error
	ds2, err := LoadData(c, DataStatusBucket, DataStatusBucket, ds)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {

		} else {
			panic(err)
		}
	}
	if ds2 == nil {
		ds = new(DataStatusStruct)
		ds.LastKnownBlock = ZeroID
		ds.LastProcessedBlock = ZeroID
		ds.LastTalliedBlockNumber = -1
	}
	DataStatus = ds
	Log.Debugf(c, "LoadDataStatus DS - %v, %v", ds, ds2)
	return ds
}

//Getters

func GetBlock(c appengine.Context, hash string) (*Block, error) {
	hash = strings.ToLower(hash)

	block, err := LoadBlock(c, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("Block %v not found", hash)
	}
	return block, nil
}

func GetBlockHeight(c appengine.Context) int {
	return LoadDataStatus(c).DBlockHeight
}

func GetDBlocksReverseOrder(c appengine.Context, start, max int) ([]*DBlock, error) {
	blocks, err := GetDBlocks(c, start, max)
	if err != nil {
		return nil, err
	}
	answer := make([]*DBlock, len(blocks))
	for i := range blocks {
		answer[len(blocks)-1-i] = blocks[i]
	}
	return answer, nil
}

func GetDBlocks(c appengine.Context, start, max int) ([]*DBlock, error) {
	answer := []*DBlock{}
	for i := start; i <= max; i++ {
		block, err := LoadDBlockBySequence(c, i)
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

func GetDBlock(c appengine.Context, keyMR string) (*DBlock, error) {
	keyMR = strings.ToLower(keyMR)

	block, err := LoadDBlock(c, keyMR)
	if err != nil {
		return nil, err
	}
	return block, nil
}

type DBInfo struct {
	BTCTxHash string
}

//Getters

func GetEntry(c appengine.Context, hash string) (*Entry, error) {
	hash = strings.ToLower(hash)
	entry, err := LoadEntry(c, hash)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		//str, _ := EncodeJSONString(Entries)
		//Log.Debugf(c, "%v not found in %v", hash, str)
		return nil, errors.New("Entry not found")
	}
	return entry, nil
}

func GetChains(c appengine.Context) ([]Chain, error) {
	//TODO: load chains from database

	answer := []Chain{}
	_, err := Datastore.QueryGetAll(c, ChainsBucket, &answer)
	if err != nil {
		Log.Errorf(c, "GetChains - %v", err)
		return nil, err
	}
	return answer, nil
}

func GetChain(c appengine.Context, hash string, startFrom, amountToFetch int) (*Chain, error) {
	Log.Debugf(c, "GetChain - %v, %v, %v", hash, startFrom, amountToFetch)
	hash = strings.ToLower(hash)
	chain, err := LoadChain(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	if chain == nil {
		return nil, errors.New("Chain not found")
	}
	entry, err := LoadEntry(c, chain.FirstEntryID)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	if entry == nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, errors.New("First entry not found")
	}
	chain.FirstEntry = entry

	entries, err := GetChainEntries(c, chain.ChainID, startFrom, amountToFetch)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	chain.Entries = entries

	return chain, nil
}

func GetChainByName(c appengine.Context, name string, startFrom, amountToFetch int) (*Chain, error) {
	Log.Debugf(c, "GetChainByName - %v, %v, %v", name, startFrom, amountToFetch)
	id, err := LoadChainIDByName(c, name)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	if id != "" {
		return GetChain(c, id, startFrom, amountToFetch)
	}

	return GetChain(c, name, startFrom, amountToFetch)
}

type EBlock struct {
	factom.EBlock
}

func GetChainEntries(c appengine.Context, chainID string, startFrom, amountToFetch int) ([]*Entry, error) {
	tmp := []Entry{}
	keys, err := Datastore.QueryGetAllKeysWithFilterLimitOffsetAndOrder(c, EntriesBucket, "ChainID=", chainID, amountToFetch, startFrom, "Timestamp", &tmp)
	if err != nil {
		return nil, err
	}
	answer := make([]*Entry, len(keys))
	for i, v := range keys {
		entry, err := GetEntry(c, v.StringID())
		if err != nil {
			return nil, err
		}
		answer[i] = entry
	}
	return answer, nil
}

func ListExternalIDs(c appengine.Context) (map[string][]string, error) {
	tmp := []Entry{}
	keys := Datastore.QueryGetAllKeysWithFilter(c, EntriesBucket, "ExternalIDs.Encoded>", "", tmp)
	answer := map[string][]string{}
	for _, v := range keys {
		entry, err := LoadEntry(c, v.StringID())
		if err != nil {
			Log.Errorf(c, "ListExternalIDs - %v", err)
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
