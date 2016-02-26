package app

import (
	"appengine"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid"
	"github.com/FactomProject/factoid/block"
	"github.com/ThePiachu/Go/Log"
)

var AnchorBlockID string = "df3ade9eec4b08d5379cc64270c30ea7315d8a8a1a69efe2b98a60ecdd69e604"
var AdminBlockID string = "000000000000000000000000000000000000000000000000000000000000000a"
var FactoidBlockID string = "000000000000000000000000000000000000000000000000000000000000000f"
var ECBlockID string = "000000000000000000000000000000000000000000000000000000000000000c"
var ZeroID string = "0000000000000000000000000000000000000000000000000000000000000000"

func SynchronizationGoroutine(c appengine.Context) {
	err := Synchronize(c)
	if err != nil {
		panic(err)
	}
	err = ProcessBlocks(c)
	if err != nil {
		panic(err)
	}
	err = TallyBalances(c)
	if err != nil {
		panic(err)
	}
}

func GetAddressInformationFromFactom(c appengine.Context, address string) (*Address, error) {
	answer := new(Address)
	answer.Address = address

	address = strings.Replace(address, "-", "", -1)
	if len(address) == 72 {
		address = address[:64]
	}

	invalid := 0 //to count how many times we got "invalid address"

	ecBalance, err := FactomdECBalance(c, address)
	Log.Debugf(c, "ECBalance - %v, %v\n\n", ecBalance, err)
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid") && !strings.Contains(err.Error(), "encoding/hex") {
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
	fctBalance, err := FactomdFactoidBalance(c, address)
	Log.Debugf(c, "FactoidBalance - %v, %v\n\n", fctBalance, err)
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid") {
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

func GetDBlockFromFactom(c appengine.Context, keyMR string) (*DBlock, error) {
	answer := new(DBlock)

	body, err := FactomdGetDBlock(c, keyMR)
	if err != nil {
		return answer, err
	}

	answer = new(DBlock)
	answer.DBHash = body.DBHash
	answer.PrevBlockKeyMR = body.Header.PrevBlockKeyMR
	answer.Timestamp = int64(body.Header.Timestamp)
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

func TallyBalances(c appengine.Context) error {
	Log.Infof(c, "TallyBalances()")
	dataStatus := LoadDataStatus(c)

	if dataStatus.LastTalliedBlockNumber == dataStatus.DBlockHeight {
		return nil
	}

	block, err := LoadDBlockBySequence(c, dataStatus.LastTalliedBlockNumber+1)
	if err != nil {
		return err
	}
	Log.Debugf(c, "Tallying block %v\n", block.SequenceNumber)
	var previousTally float64 = 0
	if dataStatus.LastTalliedBlockNumber > -1 {
		for {
			oldBlock, err := LoadDBlockBySequence(c, dataStatus.LastTalliedBlockNumber)
			if err != nil {
				panic(err)
			}
			if oldBlock.FactoidTally == "" {
				dataStatus.LastTalliedBlockNumber--
				continue
			}
			tally, err := strconv.ParseFloat(oldBlock.FactoidTally, 64)
			if err != nil {
				panic(err)
			}
			previousTally = tally
			break
		}
	}
	for {
		factoidBlock, err := LoadBlock(c, block.FactoidBlock.KeyMR)
		if err != nil {
			panic(err)
		}
		currentTally, err := strconv.ParseFloat(factoidBlock.TotalDelta, 64)
		if err != nil {
			panic(err)
		}
		previousTally += currentTally
		block.FactoidTally = fmt.Sprintf("%.8f", previousTally)
		err = SaveDBlock(c, block)
		if err != nil {
			panic(err)
		}

		dataStatus.LastTalliedBlockNumber = block.SequenceNumber
		err = SaveDataStatus(c, dataStatus)
		if err != nil {
			panic(err)
		}

		if block.SequenceNumber == dataStatus.DBlockHeight {
			return nil
		} else {
			block, err = LoadDBlockBySequence(c, block.SequenceNumber+1)
			Log.Debugf(c, "Tallying block %v\n", block.SequenceNumber)
		}
	}

	return nil
}

func ProcessBlocks(c appengine.Context) error {
	Log.Infof(c, "ProcessBlocks()")
	dataStatus := LoadDataStatus(c)
	if dataStatus.LastKnownBlock == dataStatus.LastProcessedBlock {
		return nil
	}
	toProcess := dataStatus.LastKnownBlock
	if dataStatus.ResumeProcessingFrom != "" {
		toProcess = dataStatus.ResumeProcessingFrom
	} else {
		dataStatus.NextProcessedHead = dataStatus.LastKnownBlock
	}
	previousBlock, err := LoadDBlock(c, toProcess)
	if err != nil {
		return err
	}
	for {
		block := previousBlock
		log.Printf("Processing dblock %v\n", block.KeyMR)
		toProcess = block.PrevBlockKeyMR
		if toProcess == ZeroID || block.KeyMR == dataStatus.LastProcessedBlock {
			dataStatus.LastProcessedBlock = dataStatus.NextProcessedHead
			dataStatus.NextProcessedHead = ""
			dataStatus.ResumeProcessingFrom = ""
			break
		}
		previousBlock, err = LoadDBlock(c, toProcess)

		if previousBlock.NextBlockKeyMR != "" {
			continue
		}

		blockList := block.EntryBlockList[:]
		blockList = append(blockList, block.AdminBlock)
		blockList = append(blockList, block.EntryCreditBlock)
		blockList = append(blockList, block.FactoidBlock)

		for _, v := range blockList {
			err = ProcessBlock(c, v.KeyMR)
			if err != nil {
				return err
			}
		}

		previousBlock.NextBlockKeyMR = block.KeyMR
		err = SaveDBlock(c, previousBlock)
		if err != nil {
			return err
		}

		dataStatus.ResumeProcessingFrom = block.KeyMR
		err = SaveDataStatus(c, dataStatus)
		if err != nil {
			Log.Errorf(c, "ProcessBlocks - %v", err)
			return err
		}
	}

	err = SaveDataStatus(c, dataStatus)
	if err != nil {
		Log.Errorf(c, "ProcessBlocks - %v", err)
		return err
	}
	return nil
}

func ProcessBlock(c appengine.Context, keyMR string) error {
	log.Printf("ProcessBlock()")
	previousBlock, err := LoadBlock(c, keyMR)
	if err != nil {
		return err
	}
	log.Printf("chain - %v", previousBlock.ChainID)

	for {
		block := previousBlock
		Log.Debugf(c, "Processing block %v\n", block.PartialHash)
		toProcess := block.PrevBlockHash
		if toProcess == ZeroID {
			return nil
		}

		if block.ChainID == AnchorBlockID {
			for _, v := range block.EntryList {
				err = ProcessAnchorEntry(c, v)
				if err != nil {
					return err
				}
			}
		}

		previousBlock, err = LoadBlock(c, toProcess)
		if err != nil {
			return err
		}
		if previousBlock.NextBlockHash != "" {
			return nil
		}
		previousBlock.NextBlockHash = block.PartialHash
		err = SaveBlock(c, previousBlock)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProcessAnchorEntry(c appengine.Context, e *Entry) error {
	if e.AnchorRecord == nil {
		return fmt.Errorf("No anchor record provided")
	}
	dBlock, err := LoadDBlock(c, e.AnchorRecord.KeyMR)
	if err != nil {
		return err
	}
	dBlock.AnchorRecord = e.Hash
	dBlock.AnchoredInTransaction = e.AnchorRecord.Bitcoin.TXID
	err = SaveDBlock(c, dBlock)
	if err != nil {
		return err
	}
	return nil
}

func Synchronize(c appengine.Context) error {
	Log.Infof(c, "Synchronize()")
	head, err := FactomdGetDBlockHead(c)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return err
	}
	previousKeyMR := head.KeyMR
	dataStatus := LoadDataStatus(c)
	maxHeight := dataStatus.DBlockHeight
	if maxHeight < dataStatus.NextHeight {
		maxHeight = dataStatus.NextHeight
	}
	if dataStatus.ResumeFetchingFrom != "" {
		previousKeyMR = dataStatus.ResumeFetchingFrom
	} else {
		dataStatus.NextHead = head.KeyMR
	}
	for {
		//Log.Debugf(c, "previousKeyMR - %v", previousKeyMR)
		block, err := LoadDBlock(c, previousKeyMR)
		if err != nil {
			Log.Errorf(c, "Synchronize - %v", err)
			return err
		}

		if block != nil {
			if maxHeight < block.SequenceNumber {
				maxHeight = block.SequenceNumber
			}
			if previousKeyMR == dataStatus.LastKnownBlock {
				dataStatus.LastKnownBlock = dataStatus.NextHead
				dataStatus.DBlockHeight = maxHeight
				dataStatus.NextHeight = -1
				dataStatus.ResumeFetchingFrom = ""
				dataStatus.NextHead = ""
				break
			} else {
				previousKeyMR = block.PrevBlockKeyMR
				if previousKeyMR == ZeroID {
					dataStatus.LastKnownBlock = dataStatus.NextHead
					dataStatus.DBlockHeight = maxHeight
					dataStatus.NextHeight = -1
					dataStatus.ResumeFetchingFrom = ""
					dataStatus.NextHead = ""
					break
				}
				continue
			}
		}

		body, err := GetDBlockFromFactom(c, previousKeyMR)
		if err != nil {
			Log.Errorf(c, "Error for block %v - %v", err)
			return err
		}

		Log.Debugf(c, "Processing dblock number %v", body.SequenceNumber)

		str, err := EncodeJSONString(body)
		if err != nil {
			Log.Errorf(c, "Synchronize - %v", err)
			return err
		}
		Log.Debugf(c, "%v", str)

		for _, v := range body.EntryBlockList {
			fetchedBlock, err := FetchBlock(c, v.ChainID, v.KeyMR, body.BlockTimeStr)
			if err != nil {
				Log.Errorf(c, "Synchronize - %v", err)
				return err
			}
			switch v.ChainID {
			case AdminBlockID:
				body.AdminEntries += fetchedBlock.EntryCount
				body.AdminBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			case ECBlockID:
				body.EntryCreditEntries += fetchedBlock.EntryCount
				body.EntryCreditBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			case FactoidBlockID:
				body.FactoidEntries += fetchedBlock.EntryCount
				body.FactoidBlock = ListEntry{ChainID: v.ChainID, KeyMR: v.KeyMR}
				break
			default:
				body.EntryEntries += fetchedBlock.EntryCount
				break
			}
		}
		body.EntryBlockList = body.EntryBlockList[3:]

		err = SaveDBlock(c, body)
		if err != nil {
			Log.Errorf(c, "Synchronize - %v", err)
			return err
		}

		if maxHeight < body.SequenceNumber {
			maxHeight = body.SequenceNumber
		}

		dataStatus.ResumeFetchingFrom = previousKeyMR
		dataStatus.NextHeight = maxHeight

		previousKeyMR = body.PrevBlockKeyMR
		if previousKeyMR == ZeroID {
			dataStatus.LastKnownBlock = dataStatus.NextHead
			dataStatus.DBlockHeight = maxHeight
			dataStatus.NextHeight = -1
			dataStatus.ResumeFetchingFrom = ""
			break
		}

		err = SaveDataStatus(c, dataStatus)
		if err != nil {
			Log.Errorf(c, "Synchronize - %v", err)
			return err
		}
	}
	err = SaveDataStatus(c, dataStatus)
	if err != nil {
		Log.Errorf(c, "Synchronize - %v", err)
		return err
	}
	return nil
}

func FetchBlock(c appengine.Context, chainID, hash, blockTime string) (*Block, error) {
	block := new(Block)

	raw, err := FactomdGetRaw(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	switch chainID {
	case AdminBlockID:
		block, err = ParseAdminBlock(c, chainID, hash, raw, blockTime)
		if err != nil {
			Log.Errorf(c, "Error - %v", err)
			return nil, err
		}
		break
	case ECBlockID:
		block, err = ParseEntryCreditBlock(c, chainID, hash, raw, blockTime)
		if err != nil {
			Log.Errorf(c, "Error - %v", err)
			return nil, err
		}
		break
	case FactoidBlockID:
		block, err = ParseFactoidBlock(c, chainID, hash, raw, blockTime)
		if err != nil {
			Log.Errorf(c, "Error - %v", err)
			return nil, err
		}
		break
	default:
		block, err = ParseEntryBlock(c, chainID, hash, raw, blockTime)
		if err != nil {
			Log.Errorf(c, "Error - %v", err)
			return nil, err
		}
		break
	}

	err = SaveBlock(c, block)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}

	return block, nil
}

func ParseEntryCreditBlock(c appengine.Context, chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
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

func ParseFactoidBlock(c appengine.Context, chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
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

	exchangeRate := float64(fBlock.GetExchRate())

	var ins float64
	var outs float64
	var ecs int64
	var deltas float64

	var created float64
	var destroyed float64

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

		in, err := v.TotalInputs()
		if err != nil {
			return nil, err
		}
		out, err := v.TotalOutputs()
		if err != nil {
			return nil, err
		}
		totalEcs, err := v.TotalECs()
		if err != nil {
			return nil, err
		}

		ec := uint64(float64(totalEcs) / exchangeRate)
		entry.TotalIns = factoid.ConvertDecimalToString(uint64(in))
		entry.TotalOuts = factoid.ConvertDecimalToString(uint64(out))
		entry.TotalECs = fmt.Sprintf("%d", ec)

		entry.Delta = fmt.Sprintf("%.8f", factoid.ConvertDecimalToFloat(out)-factoid.ConvertDecimalToFloat(in))

		answer.EntryList[i] = entry

		inF, err := strconv.ParseFloat(entry.TotalIns, 64)
		if err != nil {
			return nil, err
		}
		outF, err := strconv.ParseFloat(entry.TotalOuts, 64)
		if err != nil {
			return nil, err
		}
		ecF, err := strconv.ParseInt(entry.TotalECs, 10, 64)
		if err != nil {
			return nil, err
		}
		ins += inF
		outs += outF
		ecs += ecF

		deltaF, err := strconv.ParseFloat(entry.Delta, 64)
		if err != nil {
			return nil, err
		}
		if deltaF > 0.0 {
			created += deltaF
		} else {
			destroyed += deltaF
		}
		deltas += deltaF
	}

	answer.TotalIns = fmt.Sprintf("%.8f", ins)
	answer.TotalOuts = fmt.Sprintf("%.8f", outs)
	answer.TotalECs = fmt.Sprintf("%d", ecs)
	answer.Created = fmt.Sprintf("%.8f", created)
	answer.Destroyed = fmt.Sprintf("%.8f", destroyed)
	answer.TotalDelta = fmt.Sprintf("%.8f", deltas)

	answer.JSONString, err = fBlock.JSONString()
	if err != nil {
		return nil, err
	}
	answer.IsFactoidBlock = true

	return answer, nil
}

func ParseEntryBlock(c appengine.Context, chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	Log.Infof(c, "ParseEntryBlock - %x", rawBlock)
	answer := new(Block)

	eBlock := common.NewEBlock()
	_, err := eBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}

	answer.ChainID = chainID
	h, err := eBlock.KeyMR()
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	answer.PartialHash = h.String()
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	h, err = eBlock.Hash()
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	answer.FullHash = h.String()

	answer.PrevBlockHash = eBlock.Header.PrevKeyMR.String()

	answer.EntryCount = 0
	answer.EntryList = []*Entry{}
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	answer.JSONString, err = eBlock.JSONString()
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}

	Log.Debugf(c, "Block - %v", answer.JSONString)
	lastMinuteMarkedEntry := 0
	for _, v := range eBlock.Body.EBEntries {
		if IsMinuteMarker(v.String()) {
			for i := lastMinuteMarkedEntry; i < len(answer.EntryList); i++ {
				answer.EntryList[i].MinuteMarker = v.String()
			}
			lastMinuteMarkedEntry = len(answer.EntryList)
		} else {
			entry, err := FetchAndParseEntry(c, v.String(), blockTime, IsHashZeroes(answer.PrevBlockHash) && answer.EntryCount == 0)
			if err != nil {
				Log.Errorf(c, "Error - %v", err)
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
	return hash == ZeroID
}

func IsMinuteMarker(hash string) bool {
	h, err := common.HexToHash(hash)
	if err != nil {
		panic(err)
	}
	return h.IsMinuteMarker()
}

func FetchAndParseEntry(c appengine.Context, hash, blockTime string, isFirstEntry bool) (*Entry, error) {
	e := new(Entry)
	raw, err := FactomdGetRaw(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}

	entry := new(common.Entry)
	_, err = entry.UnmarshalBinaryData(raw)
	if err != nil {
		Log.Errorf(c, "Error unmarshalling data - %v, %x - %v", hash, err, raw)
		return nil, err
	}

	e.ChainID = entry.ChainID.String()
	e.Hash = hash
	str, err := entry.JSONString()
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}
	e.JSONString = str
	e.BinaryString = fmt.Sprintf("%x", raw)
	e.Timestamp = blockTime

	e.Content = ByteSliceToDecodedString(entry.Content)
	e.ExternalIDs = make([]DecodedString, len(entry.ExtIDs))
	for i, v := range entry.ExtIDs {
		e.ExternalIDs[i] = ByteSliceToDecodedString(v)
	}

	if isFirstEntry == true {
		//TODO: parse the first entry somehow perhaps?
	} else {
		if IsAnchorChainID(e.ChainID) {
			ar, err := ParseAnchorChainData(c, e.Content.Decoded)
			if err != nil {
				Log.Errorf(c, "Error - %v", err)
				return nil, err
			}
			e.AnchorRecord = ar
		}
	}

	err = SaveEntry(c, e)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		return nil, err
	}

	return e, nil
}

func IsAnchorChainID(chainID string) bool {
	return chainID == AnchorBlockID
}

func ByteSliceToDecodedStringPointer(b []byte) *DecodedString {
	ds := new(DecodedString)
	ds.Encoded = fmt.Sprintf("%x", b)
	ds.Decoded = string(b)

	if appengine.IsDevAppServer() {
		ds.Decoded = SanitizeKey(ds.Decoded)
	}
	return ds
}

func ByteSliceToDecodedString(b []byte) DecodedString {
	return *ByteSliceToDecodedStringPointer(b)
}

func ParseAdminBlock(c appengine.Context, chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
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

func GetEntriesByExtID(c appengine.Context, eid string) ([]*Entry, error) {
	ids, err := ListExternalIDs(c)
	if err != nil {
		Log.Errorf(c, "GetEntriesByExtID - %v", err)
		return nil, err
	}
	entriesToLoad := map[string]string{}

	eid = strings.ToLower(eid)

	for k, v := range ids {
		if strings.Contains(strings.ToLower(k), eid) {
			for _, id := range v {
				entriesToLoad[id] = id
			}
		}
	}

	answer := []*Entry{}

	for _, v := range entriesToLoad {
		entry, err := LoadEntry(c, v)
		if err != nil {
			Log.Errorf(c, "GetEntriesByExtID - %v", err)
			return nil, err
		}
		answer = append(answer, entry)
	}

	return answer, nil
}
