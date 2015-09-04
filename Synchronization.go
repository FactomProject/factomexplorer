package main

import (
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid"
	"github.com/FactomProject/factoid/block"
	"github.com/FactomProject/factom"
	"github.com/FactomProject/fctwallet/Wallet"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var AnchorBlockID string

func init() {
	AnchorBlockID = ReadConfig().Anchor.AnchorChainID
}

func Log(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	fmt.Printf(file+":"+strconv.Itoa(line)+" - "+format+"\n", args...)
}

func GetAddressInformationFromFactom(address string) (*Address, error) {
	answer := new(Address)
	answer.Address = address

	address = strings.Replace(address, "-", "", -1)
	if len(address) == 72 {
		address = address[:64]
	}

	invalid := 0 //to count how many times we got "invalid address"

	ecBalance, err := Wallet.ECBalance(address)
	fmt.Printf("ECBalance - %v, %v\n\n", ecBalance, err)
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid name or address") && !strings.Contains(err.Error(), "encoding/hex") {
			return nil, err
		}
		invalid++
	} else {
		answer.Balance = fmt.Sprintf("%d", ecBalance)
		answer.AddressType = "EC Address"
		if ecBalance > 0 {
			return answer, nil
		}
	}
	fctBalance, err := Wallet.FactoidBalance(address)
	fmt.Printf("FactoidBalance - %v, %v\n\n", fctBalance, err)
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid name or address") {
			return nil, err
		}
		invalid++
	} else {
		answer.AddressType = "Factoid Address"
		if fctBalance > 0 {
			answer.Balance = factoid.ConvertDecimalToString(uint64(fctBalance))
			return answer, nil
		}
	}
	if invalid > 1 {
		//2 responses - it's not a valid address period
		return nil, fmt.Errorf("Invalid address")
	}
	if invalid == 0 {
		//no invalid responses - meaning it's a public key valid for both factoid and ec
		answer.AddressType = "Unknown Address"
	}

	return answer, nil
}

func GetDBlockFromFactom(keyMR string) (*DBlock, error) {
	answer := new(DBlock)

	body, err := factom.GetDBlock(keyMR)
	if err != nil {
		return answer, err
	}

	answer = new(DBlock)
	answer.DBHash = body.DBHash
	answer.PrevBlockKeyMR = body.Header.PrevBlockKeyMR
	answer.Timestamp = body.Header.Timestamp
	answer.SequenceNumber = body.Header.SequenceNumber
	answer.EntryBlockList = make([]ListEntry, len(body.EntryBlockList))
	for i, v := range body.EntryBlockList {
		answer.EntryBlockList[i].ChainID = v.ChainID
		answer.EntryBlockList[i].KeyMR = v.KeyMR
	}
	//answer.DBlock = *body
	answer.BlockTimeStr = TimestampToString(body.Header.Timestamp)
	answer.KeyMR = keyMR

	return answer, nil
}

func TimestampToString(timestamp uint64) string {
	blockTime := time.Unix(int64(timestamp), 0)
	return blockTime.Format("2006-01-02 15:04:05")
}

func ProcessBlocks() error {
	log.Println("ProcessBlocks()")
	dataStatus := LoadDataStatus()
	if dataStatus.LastKnownBlock == dataStatus.LastProcessedBlock {
		return nil
	}
	toProcess := dataStatus.LastKnownBlock
	previousBlock, err := LoadDBlock(toProcess)
	if err != nil {
		return err
	}
	for {
		block := previousBlock
		log.Printf("Processing dblock %v\n", block.KeyMR)
		toProcess = block.PrevBlockKeyMR
		if toProcess == "0000000000000000000000000000000000000000000000000000000000000000" || block.KeyMR == dataStatus.LastProcessedBlock {
			dataStatus.LastProcessedBlock = dataStatus.LastKnownBlock
			break
		}
		previousBlock, err = LoadDBlock(toProcess)

		if previousBlock.NextBlockKeyMR != "" {
			continue
		}

		blockList := block.EntryBlockList[:]
		blockList = append(blockList, block.AdminBlock)
		blockList = append(blockList, block.EntryCreditBlock)
		blockList = append(blockList, block.FactoidBlock)

		for _, v := range blockList {
			err = ProcessBlock(v.KeyMR)
			if err != nil {
				return err
			}
		}

		previousBlock.NextBlockKeyMR = block.KeyMR
		err = SaveDBlock(previousBlock)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProcessBlock(keyMR string) error {
	log.Printf("ProcessBlock()")
	previousBlock, err := LoadBlock(keyMR)
	if err != nil {
		return err
	}
	log.Printf("chain - %v", previousBlock.ChainID)

	for {
		block := previousBlock
		log.Printf("Processing block %v\n", block.PartialHash)
		toProcess := block.PrevBlockHash
		if toProcess == "0000000000000000000000000000000000000000000000000000000000000000" {
			return nil
		}

		if block.ChainID == AnchorBlockID {
			for _, v := range block.EntryList {
				err = ProcessAnchorEntry(v)
				if err != nil {
					return err
				}
			}
		}

		previousBlock, err = LoadBlock(toProcess)
		if err != nil {
			return err
		}
		if previousBlock.NextBlockHash != "" {
			return nil
		}
		previousBlock.NextBlockHash = block.PartialHash
		err = SaveBlock(previousBlock)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProcessAnchorEntry(e *Entry) error {
	if e.AnchorRecord == nil {
		return fmt.Errorf("No anchor record provided")
	}
	dBlock, err := LoadDBlock(e.AnchorRecord.KeyMR)
	if err != nil {
		return err
	}
	dBlock.AnchorRecord = e.Hash
	dBlock.AnchoredInTransaction = e.AnchorRecord.Bitcoin.TXID
	err = SaveDBlock(dBlock)
	if err != nil {
		return err
	}
	return nil
}

func Synchronize() error {
	log.Println("Synchronize()")
	head, err := factom.GetDBlockHead()
	if err != nil {
		Log("Error - %v", err)
		return err
	}
	previousKeyMR := head.KeyMR
	dataStatus := LoadDataStatus()
	maxHeight := dataStatus.DBlockHeight
	for {

		block, err := LoadDBlock(previousKeyMR)
		if err != nil {
			Log("Error - %v", err)
			return err
		}

		if block != nil {
			if maxHeight < block.SequenceNumber {
				maxHeight = block.SequenceNumber
			}
			if previousKeyMR == dataStatus.LastKnownBlock {
				dataStatus.LastKnownBlock = head.KeyMR
				dataStatus.DBlockHeight = maxHeight
				break
			} else {
				previousKeyMR = block.PrevBlockKeyMR
				continue
			}
		}
		body, err := GetDBlockFromFactom(previousKeyMR)
		if err != nil {
			Log("Error - %v", err)
			return err
		}

		log.Printf("\n\nProcessing dblock number %v\n", body.SequenceNumber)

		str, err := EncodeJSONString(body)
		if err != nil {
			Log("Error - %v", err)
			return err
		}
		log.Printf("%v", str)

		for _, v := range body.EntryBlockList {
			fetchedBlock, err := FetchBlock(v.ChainID, v.KeyMR, body.BlockTimeStr)
			if err != nil {
				Log("Error - %v", err)
				return err
			}
			switch v.ChainID {
			case "000000000000000000000000000000000000000000000000000000000000000a":
				body.AdminEntries += fetchedBlock.EntryCount
				body.AdminBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			case "000000000000000000000000000000000000000000000000000000000000000c":
				body.EntryCreditEntries += fetchedBlock.EntryCount
				body.EntryCreditBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			case "000000000000000000000000000000000000000000000000000000000000000f":
				body.FactoidEntries += fetchedBlock.EntryCount
				body.FactoidBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			default:
				body.EntryEntries += fetchedBlock.EntryCount
				break
			}
		}
		body.EntryBlockList = body.EntryBlockList[3:]

		err = SaveDBlock(body)
		if err != nil {
			Log("Error - %v", err)
			return err
		}

		if maxHeight < body.SequenceNumber {
			maxHeight = body.SequenceNumber
		}
		previousKeyMR = body.PrevBlockKeyMR
		if previousKeyMR == "0000000000000000000000000000000000000000000000000000000000000000" {
			dataStatus.LastKnownBlock = head.KeyMR
			dataStatus.DBlockHeight = maxHeight
			break
		}

	}
	err = SaveDataStatus(dataStatus)
	if err != nil {
		Log("Error - %v", err)
		return err
	}
	return nil
}

func FetchBlock(chainID, hash, blockTime string) (*Block, error) {
	block := new(Block)

	raw, err := factom.GetRaw(hash)
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}
	switch chainID {
	case "000000000000000000000000000000000000000000000000000000000000000a":
		block, err = ParseAdminBlock(chainID, hash, raw, blockTime)
		if err != nil {
			Log("Error - %v", err)
			return nil, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000c":
		block, err = ParseEntryCreditBlock(chainID, hash, raw, blockTime)
		if err != nil {
			Log("Error - %v", err)
			return nil, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000f":
		block, err = ParseFactoidBlock(chainID, hash, raw, blockTime)
		if err != nil {
			Log("Error - %v", err)
			return nil, err
		}
		break
	default:
		block, err = ParseEntryBlock(chainID, hash, raw, blockTime)
		if err != nil {
			Log("Error - %v", err)
			return nil, err
		}
		break
	}

	err = SaveBlock(block)
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}

	return block, nil
}

func ParseEntryCreditBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer := new(Block)

	ecBlock := common.NewECBlock()
	_, err := ecBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return nil, err
	}

	answer.ChainID = chainID
	h, err := ecBlock.Hash()
	if err != nil {
		return nil, err
	}
	answer.FullHash = h.String()

	h, err = ecBlock.HeaderHash()
	if err != nil {
		return nil, err
	}
	answer.PartialHash = h.String()

	answer.PrevBlockHash = ecBlock.Header.PrevLedgerKeyMR.String()

	answer.EntryCount = len(ecBlock.Body.Entries)
	answer.EntryList = make([]*Entry, answer.EntryCount)

	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	for i, v := range ecBlock.Body.Entries {
		entry := new(Entry)

		marshalled, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.Hash = v.Hash().String()

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return nil, err
		}
		entry.ShortEntry = v.Interpret()

		answer.EntryList[i] = entry
	}

	answer.JSONString, err = ecBlock.JSONString()
	if err != nil {
		return nil, err
	}
	answer.IsEntryCreditBlock = true

	return answer, nil
}

func ParseFactoidBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer := new(Block)

	fBlock := new(block.FBlock)
	_, err := fBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return nil, err
	}

	answer.ChainID = chainID
	answer.PartialHash = fBlock.GetHash().String()
	answer.FullHash = fBlock.GetLedgerKeyMR().String()
	answer.PrevBlockHash = fmt.Sprintf("%x", fBlock.PrevKeyMR.Bytes())

	transactions := fBlock.GetTransactions()
	answer.EntryCount = len(transactions)
	answer.EntryList = make([]*Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	for i, v := range transactions {
		entry := new(Entry)
		bin, err := v.MarshalBinary()

		if err != nil {
			return nil, err
		}

		entry.BinaryString = fmt.Sprintf("%x", bin)
		entry.Timestamp = TimestampToString(v.GetMilliTimestamp() / 1000)
		entry.Hash = v.GetHash().String()
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return nil, err
		}

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = fBlock.JSONString()
	if err != nil {
		return nil, err
	}
	answer.IsFactoidBlock = true

	return answer, nil
}

func ParseEntryBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	Log("ParseEntryBlock - %x", rawBlock)
	answer := new(Block)

	eBlock := common.NewEBlock()
	_, err := eBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}

	answer.ChainID = chainID
	h, err := eBlock.KeyMR()
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}
	answer.PartialHash = h.String()
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}
	h, err = eBlock.Hash()
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}
	answer.FullHash = h.String()

	answer.PrevBlockHash = eBlock.Header.PrevKeyMR.String()

	answer.EntryCount = 0
	answer.EntryList = []*Entry{}
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	answer.JSONString, err = eBlock.JSONString()
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}

	Log("Block - %v", answer.JSONString)
	lastMinuteMarkedEntry := 0
	for _, v := range eBlock.Body.EBEntries {
		if IsMinuteMarker(v.String()) {
			for i := lastMinuteMarkedEntry; i < len(answer.EntryList); i++ {
				answer.EntryList[i].MinuteMarker = v.String()
			}
			lastMinuteMarkedEntry = len(answer.EntryList)
		} else {
			entry, err := FetchAndParseEntry(v.String(), blockTime, IsHashZeroes(answer.PrevBlockHash) && answer.EntryCount == 0)
			if err != nil {
				Log("Error - %v", err)
				return nil, err
			}
			answer.EntryCount++
			answer.EntryList = append(answer.EntryList, entry)
		}
	}

	answer.IsEntryBlock = true

	return answer, nil
}

func IsHashZeroes(hash string) bool {
	return hash == "0000000000000000000000000000000000000000000000000000000000000000"
}

func IsMinuteMarker(hash string) bool {
	h, err := common.HexToHash(hash)
	if err != nil {
		panic(err)
	}
	return h.IsMinuteMarker()
}

func FetchAndParseEntry(hash, blockTime string, isFirstEntry bool) (*Entry, error) {
	e := new(Entry)
	raw, err := factom.GetRaw(hash)
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}

	entry := new(common.Entry)
	_, err = entry.UnmarshalBinaryData(raw)
	if err != nil {
		Log("Error unmarshalling data - %v, %x - %v", hash, err, raw)
		return nil, err
	}

	e.ChainID = entry.ChainID.String()
	e.Hash = hash
	str, err := entry.JSONString()
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}
	e.JSONString = str
	e.BinaryString = fmt.Sprintf("%x", raw)
	e.Timestamp = blockTime

	e.Content = ByteSliceToDecodedStringPointer(entry.Content)
	e.ExternalIDs = make([]DecodedString, len(entry.ExtIDs))
	for i, v := range entry.ExtIDs {
		e.ExternalIDs[i] = ByteSliceToDecodedString(v)
	}

	if isFirstEntry == true {
		//TODO: parse the first entry somehow perhaps?
	} else {
		if IsAnchorChainID(e.ChainID) {
			ar, err := ParseAnchorChainData(e.Content.Decoded)
			if err != nil {
				Log("Error - %v", err)
				return nil, err
			}
			e.AnchorRecord = ar
		}
	}

	err = SaveEntry(e)
	if err != nil {
		Log("Error - %v", err)
		return nil, err
	}

	return e, nil
}

func IsAnchorChainID(chainID string) bool {
	return chainID == AnchorBlockID
}

func ParseAnchorChainData(data string) (*AnchorRecord, error) {
	if len(data) < 128 {
		return nil, fmt.Errorf("Data too short")
	}
	tmp := data[:len(data)-128]
	ar := new(AnchorRecord)

	err := common.DecodeJSONString(tmp, ar)
	if err != nil {
		Log("ParseAnchorChainData - %v", err)
		return nil, err
	}
	return ar, nil
}

func ByteSliceToDecodedStringPointer(b []byte) *DecodedString {
	ds := new(DecodedString)
	ds.Encoded = fmt.Sprintf("%x", b)
	ds.Decoded = string(b)
	return ds
}

func ByteSliceToDecodedString(b []byte) DecodedString {
	return *ByteSliceToDecodedStringPointer(b)
}

func ParseAdminBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer := new(Block)

	aBlock := new(common.AdminBlock)
	_, err := aBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return nil, err
	}

	answer.ChainID = chainID
	fullHash, err := aBlock.LedgerKeyMR()
	if err != nil {
		return nil, err
	}
	answer.FullHash = fullHash.String()
	partialHash, err := aBlock.PartialHash()
	if err != nil {
		return nil, err
	}
	answer.PartialHash = partialHash.String()
	answer.EntryCount = len(aBlock.ABEntries)
	answer.PrevBlockHash = fmt.Sprintf("%x", aBlock.Header.PrevLedgerKeyMR.GetBytes())
	answer.EntryList = make([]*Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	for i, v := range aBlock.ABEntries {
		marshalled, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		entry := new(Entry)

		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Hash = v.Hash().String()
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return nil, err
		}
		entry.ShortEntry = v.Interpret()

		answer.EntryList[i] = entry

	}
	answer.JSONString, err = aBlock.JSONString()
	if err != nil {
		return nil, err
	}

	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	answer.IsAdminBlock = true

	return answer, nil
}
