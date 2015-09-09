package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/FactomProject/factom"
)

var (
	_ = fmt.Sprint("tmp")
	_ = os.Stdout
	_ = io.EOF
	_ = log.Fatal
)

var tpl *template.Template

const jsondblock = `{"Header":{"EntryCount":2,"BlockID":1,"PrevBlockHash":"74c052d99050a334d35e6cbf196e2242921140308e12f500bb73298622f7395d","MerkleRoot":"2d7cb0911ede2948eec478d18e59baa50b5ccb1fa225a8c71b056f77bdb0df6b","Version":1,"TimeStamp":1424298507,"BatchFlag":0},"DBEntries":[{"MerkleRoot":"0538abefba2bb9c96b75698c1d18a2c32e0fed88bc1e1deac8901963697dbd69","ChainID":"f4f614fd9b59fe26827137937d401e2b82125c4eb48f966e6e8d30f187184cb0"},{"MerkleRoot":"8ff0698ebcb2d034d52b583f8b3646d4a9e992867fbe77ebbfea2d8c0d0d8547","ChainID":"0100000000000000000000000000000000000000000000000000000000000000"}]}`

func handleDBlock (w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleDBlock")
	dblock := new(factom.DBlock)
	
	err := json.Unmarshal([]byte(jsondblock), dblock)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(dblock)
	
	tpl.Execute(w, dblock)
}

func TestHandle(t *testing.T) {
	tpl, _ = template.ParseFiles("views/tmp.html")
	handleStatic := http.FileServer(http.Dir("./"))
	
//	http.Handle("/css/", http.StripPrefix("/css/",
//		http.FileServer(http.Dir("./css"))))
	http.Handle("/css/", handleStatic)
	http.Handle("/fonts/", handleStatic)
	http.Handle("/images/", handleStatic)
	http.Handle("/scripts/", handleStatic)
	http.Handle("/", handleStatic)
	http.HandleFunc("/dblock/", handleDBlock)
	http.ListenAndServe(":8087", nil)
}