package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/FactomProject/factom"
)

var (
	_ = fmt.Sprint("tmp")
	_ = os.Stdout
)

const jsonblocks = `{"Header":{"BlockID":0,"PrevBlockHash":"0000000000000000000000000000000000000000000000000000000000000000","MerkleRoot":"29a960e1e98fe3881cbe96b18498fbab0bdda5fc3c5e13fc465a8d2ee33e2b1e","Version":1,"TimeStamp":1424298447,"BatchFlag":0,"EntryCount":2},"DBEntries":[{"MerkleRoot":"2d8fc252e8ce40ee7ff0396621f69854e3f058fe640533510b81b89e9b68408d","ChainID":"f4f614fd9b59fe26827137937d401e2b82125c4eb48f966e6e8d30f187184cb0"},{"MerkleRoot":"2cc92b09333c7d6172939be031332a12ca5a47b4716f4e8fcbfd69435a394e6f","ChainID":"0100000000000000000000000000000000000000000000000000000000000000"}]}
{"Header":{"PrevBlockHash":"74c052d99050a334d35e6cbf196e2242921140308e12f500bb73298622f7395d","MerkleRoot":"2d7cb0911ede2948eec478d18e59baa50b5ccb1fa225a8c71b056f77bdb0df6b","Version":1,"TimeStamp":1424298507,"BatchFlag":0,"EntryCount":2,"BlockID":1},"DBEntries":[{"MerkleRoot":"0538abefba2bb9c96b75698c1d18a2c32e0fed88bc1e1deac8901963697dbd69","ChainID":"f4f614fd9b59fe26827137937d401e2b82125c4eb48f966e6e8d30f187184cb0"},{"MerkleRoot":"8ff0698ebcb2d034d52b583f8b3646d4a9e992867fbe77ebbfea2d8c0d0d8547","ChainID":"0100000000000000000000000000000000000000000000000000000000000000"}]}
{"Header":{"TimeStamp":1424300127,"BatchFlag":0,"EntryCount":2,"BlockID":2,"PrevBlockHash":"4053288e28b19c46b622ad966c0a416430389c217031df839cda7b2f8d758dcc","MerkleRoot":"aa8237aa27e4e5e7d11e80e090a50eaf82071e4db94f409d45138c9e31b41a44","Version":1},"DBEntries":[{"MerkleRoot":"0a0d44834cf602909b9f71363fb2d8d22e054731864de5b46e886d45c11159c8","ChainID":"f4f614fd9b59fe26827137937d401e2b82125c4eb48f966e6e8d30f187184cb0"},{"MerkleRoot":"24535707e9f50f7e14a728e91c543a6fc154c2f76a82a83583a35b215b9781ec","ChainID":"0100000000000000000000000000000000000000000000000000000000000000"}]}`

func FakeGetDBlocks(from, to int) ([]factom.DBlock, error) {
	var _ = from
	var _ = to
	
	dblocks := make([]factom.DBlock, 0)
	
	dec := json.NewDecoder(strings.NewReader(jsonblocks))
	for {
		var block factom.DBlock
		if err := dec.Decode(&block); err == io.EOF {
			break
		} else if err != nil {
			return dblocks, err
		}
		dblocks = append(dblocks, block)
	}

	return dblocks, nil
}

func FakehandleDBlocks(w http.ResponseWriter, r *http.Request) {
//	height, err := factom.GetBlockHeight()
//	if err != nil {
//		log.Fatal(err)
//	}
	dBlocks, err := FakeGetDBlocks(0, 1)
	if err != nil {
		log.Fatal(err)
	}
	if dBlocks == nil {
		log.Fatal("dBlocks is nil")
	}
	
	fmt.Printf("%#v\n", dBlocks)

	t := template.Must(template.ParseFiles("views/tmp.html"))
	if err != nil {
		log.Fatal("Something went wrong with ParseFiles ", err)
	}
	t.ExecuteTemplate(w, "tmp.html", dBlocks)
}

func TestDBlocks(t *testing.T) {
//	http.Handle("/css/", http.StripPrefix("/css/",
//		http.FileServer(http.Dir("./css"))))
	http.Handle("/css/", http.FileServer(http.Dir("./")))
	http.Handle("/fonts/", http.FileServer(http.Dir("./")))
	http.Handle("/images/", http.FileServer(http.Dir("./")))
	http.Handle("/scripts/", http.FileServer(http.Dir("./")))
	http.HandleFunc("/", FakehandleDBlocks)
	http.HandleFunc("/dblocks/", FakehandleDBlocks)
	http.ListenAndServe(":8087", nil)
}