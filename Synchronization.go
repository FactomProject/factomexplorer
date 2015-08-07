package main

import (
	"fmt"
	"github.com/FactomProject/FactomCode/common"
	"github.com/FactomProject/factoid/block"
	"github.com/FactomProject/factom"
	"log"
	"time"
)

func GetDBlockFromFactom(keyMR string) (*DBlock, error) {
	answer:=new(DBlock)

	body, err := factom.GetDBlock(keyMR)
	if err != nil {
		return answer, err
	}

	answer = new(DBlock)
	answer.DBlock = *body
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
	dataStatus:=LoadDataStatus()
	maxHeight:=dataStatus.DBlockHeight
	for {

		block, err := LoadDBlock(previousKeyMR)
		if err!=nil {
			return err
		}

		if block!=nil {
			if maxHeight < block.DBlock.Header.SequenceNumber {
				maxHeight = block.DBlock.Header.SequenceNumber
			}
			if previousKeyMR == dataStatus.LastKnownBlock {
				dataStatus.LastKnownBlock = head.KeyMR
				dataStatus.DBlockHeight = maxHeight
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

		err = SaveDBlock(body)
		if err != nil {
			return err
		}

		previousKeyMR = body.Header.PrevBlockKeyMR
		if previousKeyMR == "0000000000000000000000000000000000000000000000000000000000000000" {
			dataStatus.LastKnownBlock = head.KeyMR
			dataStatus.DBlockHeight = maxHeight
			break
		}

	}
	err=SaveDataStatus(dataStatus)
	if err!=nil {
		return err
	}
	return nil
}


func FetchBlock(chainID, hash, blockTime string) (*Block, error) {
	block:=new(Block)

	raw, err := factom.GetRaw(hash)
	if err != nil {
		return nil, err
	}
	switch chainID {
	case "000000000000000000000000000000000000000000000000000000000000000a":
		block, err = ParseAdminBlock(chainID, hash, raw, blockTime)
		if err != nil {
		return nil, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000c":
		block, err = ParseEntryCreditBlock(chainID, hash, raw, blockTime)
		if err != nil {
		return nil, err
		}
		break
	case "000000000000000000000000000000000000000000000000000000000000000f":
		block, err = ParseFactoidBlock(chainID, hash, raw, blockTime)
		if err != nil {
		return nil, err
		}
		break
	default:
		block, err = ParseEntryBlock(chainID, hash, raw, blockTime)
		if err != nil {
		return nil, err
		}
		break
	}

	err = SaveBlock(block)
	if err != nil {
		return nil, err
	}


	return block, nil
}


func ParseEntryCreditBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer:=new(Block)

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
		entry:=new(Entry)

		marshalled, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.Hash = fmt.Sprintf("%x", v.ECID())

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return nil, err
		}
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}

	answer.JSONString, err = ecBlock.JSONString()
	if err != nil {
			return nil, err
	}
	answer.SpewString = ecBlock.Spew()
	answer.IsEntryCreditBlock = true

	return answer, nil
}

func ParseFactoidBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer:=new(Block)

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
		entry:=new(Entry)
		bin, err:=v.MarshalBinary()

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
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = fBlock.JSONString()
	if err != nil {
			return nil, err
	}
	answer.SpewString = fBlock.Spew()
	answer.IsFactoidBlock = true

	return answer, nil
}

func ParseEntryBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer:=new(Block)

	eBlock := common.NewEBlock()
	_, err := eBlock.UnmarshalBinaryData(rawBlock)
	if err != nil {
		return nil, err
	}

	answer.ChainID = chainID
	h, err:=eBlock.KeyMR()
	if err != nil {
		return nil, err
	}
	answer.PartialHash = h.String()
	if err != nil {
		return nil, err
	}
	h, err=eBlock.Hash()
	if err != nil {
		return nil, err
	}
	answer.FullHash = h.String()

	answer.PrevBlockHash = eBlock.Header.PrevKeyMR.String()

	answer.EntryCount = len(eBlock.Body.EBEntries)
	answer.EntryList = make([]*Entry, answer.EntryCount)
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)

	for i, v := range eBlock.Body.EBEntries {
		entry, err:=FetchAndParseEntry(v.String(), blockTime)
		if err != nil {
			return nil, err
		}

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = eBlock.JSONString()
	if err != nil {
		return nil, err
	}
	answer.SpewString = eBlock.Spew()

	answer.IsEntryBlock = true

	return answer, nil
}

func FetchAndParseEntry(hash, blockTime string) (*Entry, error) {
	e:=new(Entry)
	raw, err:=factom.GetRaw(hash)
	if err!=nil {
		return nil, err
	}

	entry:=new(common.Entry)
	_, err = entry.UnmarshalBinaryData(raw)
	if err!=nil {
		return nil, err
	}


	e.ChainID = entry.ChainID.String()
	e.Hash = hash
	str, err:=entry.JSONString()
	if err!=nil {
		return nil, err
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

	err = SaveEntry(e)
	if err!=nil {
		return nil, err
	}

	return e, nil
}

func ByteSliceToDecodedString(b []byte) (DecodedString) {
	var ds DecodedString
	ds.Encoded = fmt.Sprintf("%x", b)
	ds.Decoded = string(b)
	return ds
}

func ParseAdminBlock(chainID, hash string, rawBlock []byte, blockTime string) (*Block, error) {
	answer:=new(Block)

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
		entry:=new(Entry)

		entry.BinaryString = fmt.Sprintf("%x", marshalled)
		entry.Hash = fmt.Sprintf("%x", marshalled)
		entry.Timestamp = blockTime
		entry.ChainID = chainID

		entry.JSONString, err = v.JSONString()
		if err != nil {
			return nil, err
		}
		entry.SpewString = v.Spew()

		answer.EntryList[i] = entry
	}
	answer.JSONString, err = aBlock.JSONString()
	if err != nil {
			return nil, err
	}

	answer.SpewString = aBlock.Spew()
	answer.BinaryString = fmt.Sprintf("%x", rawBlock)
	answer.IsAdminBlock = true

	return answer, nil
}