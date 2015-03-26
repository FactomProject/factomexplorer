package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/FactomProject/factom"
)

var _ = fmt.Sprint("tmp")

var (
	handleStatic = http.FileServer(http.Dir("./"))
	tpl = new(template.Template)
)

func init() {
	tpl = template.Must(template.ParseFiles(
		"views/index.html",
		"views/dblock.html",
		"views/eblock.html",
		"views/sentry.html",
	))
	http.Handle("/css/", handleStatic)
	http.Handle("/fonts/", handleStatic)
	http.Handle("/images/", handleStatic)
	http.Handle("/scripts/", handleStatic)
	http.HandleFunc("/", handleDBlocks)
	http.HandleFunc("/dblocks/", handleDBlocks)
	http.HandleFunc("/dblock/", handleDBlock)
	http.HandleFunc("/eblock/", handleEBlock)
	http.HandleFunc("/sentry/", handleEntry)
}

func main() {
	http.ListenAndServe(":8087", nil)
}

func handleDBlocks(w http.ResponseWriter, r *http.Request) {
	height, err := factom.GetBlockHeight()
	if err != nil {
		log.Fatal(err)
	}
	dBlocks, err := factom.GetDBlocks(0, height)
	if err != nil {
		log.Fatal(err)
	}
	if dBlocks == nil {
		log.Fatal("dBlocks is nil")
	}

	tpl.ExecuteTemplate(w, "index.html", dBlocks)
}

func handleDBlock(w http.ResponseWriter, r *http.Request) {
	mr := r.URL.Path[len("/dblock/"):]
	
	dblock, err := factom.GetDBlock(mr)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(w, "dblock.html", dblock)
}

func handleEBlock(w http.ResponseWriter, r *http.Request) {
	mr := r.URL.Path[len("/eblock/"):]
	
	eblock, err := factom.GetEBlock(mr)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(w, "eblock.html", eblock)
}

func handleEntry(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Path[len("/entry/"):]
	
	entry, err := factom.GetEntry(hash)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(w, "sentry.html", entry)
}
