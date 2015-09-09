package app

import (
	"appengine"
	"appengine/datastore"
	"bytes"
	"errors"
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factom"
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

var BucketList []string = []string{DBlocksBucket, DBlockKeyMRsBySequenceBucket, BlocksBucket, EntriesBucket, ChainsBucket, ChainIDsByEncodedNameBucket, ChainIDsByDecodedNameBucket, BlockIndexesBucket, DataStatusBucket}

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

	EntryList []*Entry

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
	Content     *DecodedString

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

func RecordChain(c appengine.Context, block *Block) error {
	if block.PrevBlockHash != ZeroID {
		return nil
	}

	chain := new(Chain)
	chain.ChainID = block.ChainID
	chain.FirstEntryID = block.EntryList[0].Hash
	chain.Names = block.EntryList[0].ExternalIDs[:]

	err := SaveChain(c, chain)
	if err != nil {
		return err
	}

	Log.Debugf(c, "\n\nChain - %v\n\n", chain)
	return nil
}

func StoreEntriesFromBlock(c appengine.Context, block *Block) error {
	for _, v := range block.EntryList {
		err := SaveEntry(c, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadDBlockKeyMRBySequence(c appengine.Context, sequence int) (string, error) {
	seq := fmt.Sprintf("%v", sequence)

	key := new(string)
	key2, err := LoadData(c, DBlockKeyMRsBySequenceBucket, seq, key)
	if err != nil {
		return "", err
	}
	if key2 == nil {
		return "", nil
	}
	return *key, nil
}

func SaveDBlockKeyMRBySequence(c appengine.Context, keyMR string, sequence int) error {
	seq := fmt.Sprintf("%v", sequence)
	err := SaveData(c, DBlockKeyMRsBySequenceBucket, seq, keyMR)
	if err != nil {
		return err
	}
	return nil
}

//Savers and Loaders
func SaveDBlock(c appengine.Context, b *DBlock) error {
	err := SaveData(c, DBlocksBucket, b.KeyMR, b)
	if err != nil {
		return err
	}

	err = SaveDBlockKeyMRBySequence(c, b.KeyMR, b.SequenceNumber)
	if err != nil {
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

func SaveBlockIndex(c appengine.Context, index, hash string) error {
	err := SaveData(c, BlockIndexesBucket, index, hash)
	if err != nil {
		return err
	}

	return nil
}

func LoadBlockIndex(c appengine.Context, hash string) (string, error) {
	ind := new(string)
	ind2, err := LoadData(c, BlockIndexesBucket, hash, ind)
	if err != nil {
		return "", err
	}
	if ind2 == nil {
		return "", nil
	}

	return *ind, nil
}

func SaveBlock(c appengine.Context, b *Block) error {
	StoreEntriesFromBlock(c, b)

	err := SaveBlockIndex(c, b.FullHash, b.PartialHash)
	if err != nil {
		return err
	}
	err = SaveBlockIndex(c, b.PartialHash, b.PartialHash)
	if err != nil {
		return err
	}

	err = SaveData(c, BlocksBucket, b.PartialHash, b)
	if err != nil {
		return err
	}

	if b.IsEntryBlock {
		RecordChain(c, b)
	}

	return nil
}

func LoadBlock(c appengine.Context, hash string) (*Block, error) {
	key, err := LoadBlockIndex(c, hash)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}

	block := new(Block)
	block2, err := LoadData(c, BlocksBucket, key, block)
	if err != nil {
		return nil, err
	}
	if block2 == nil {
		return nil, nil
	}

	return block, nil
}

func SaveEntry(c appengine.Context, e *Entry) error {
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

	return entry, nil
}

func SaveChainIDsByName(c appengine.Context, chainID, decodedName, encodedName string) error {
	err := SaveData(c, ChainIDsByDecodedNameBucket, decodedName, chainID)
	if err != nil {
		return err
	}

	err = SaveData(c, ChainIDsByEncodedNameBucket, encodedName, chainID)
	if err != nil {
		return err
	}

	return nil
}

func LoadChainIDByName(c appengine.Context, name string) (string, error) {
	entry := new(string)
	entry2, err := LoadData(c, ChainIDsByDecodedNameBucket, name, entry)
	if err != nil {
		return "", err
	}
	if entry2 != nil {
		return *entry, nil
	}

	entry = new(string)
	entry2, err = LoadData(c, ChainIDsByEncodedNameBucket, name, entry)
	if err != nil {
		return "", err
	}
	if entry2 != nil {
		return *entry, nil
	}

	return "", nil
}

func SaveChain(c appengine.Context, chain *Chain) error {
	err := SaveData(c, ChainsBucket, chain.ChainID, chain)
	if err != nil {
		return err
	}

	for _, v := range chain.Names {
		err = SaveChainIDsByName(c, chain.ChainID, v.Decoded, v.Encoded)
		if err != nil {
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

func GetChains(c appengine.Context) ([]*Chain, error) {
	//TODO: load chains from database
	/*answer := []*Chain{}
	for _, v := range Chains {
		answer = append(answer, v)
	}
	return answer, nil*/

	//TODO: FIXME: do
	return nil, nil
}

func GetChain(c appengine.Context, hash string) (*Chain, error) {
	hash = strings.ToLower(hash)
	chain, err := LoadChain(c, hash)
	if err != nil {
		return nil, err
	}
	if chain == nil {
		return chain, errors.New("Chain not found")
	}
	entry, err := LoadEntry(c, chain.FirstEntryID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return chain, errors.New("First entry not found")
	}
	chain.FirstEntry = entry
	return chain, nil
}

func GetChainByName(c appengine.Context, name string) (*Chain, error) {
	id, err := LoadChainIDByName(c, name)
	if err != nil {
		return nil, err
	}
	if id != "" {
		return GetChain(c, id)
	}

	return GetChain(c, name)
}

type EBlock struct {
	factom.EBlock
}
