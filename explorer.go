// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/FactomProject/factom"
	"github.com/hoisie/web"
)

var _ = fmt.Sprint("tmp")

var (
	cfg      = ReadConfig().Explorer
	server   = web.NewServer()
	tpl      = new(template.Template)
	ExtIDMap map[string]bool
)

func main() {
	var err error
	
	server.Config.StaticDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.StaticDir != "" {
		server.Config.StaticDir = cfg.StaticDir
	}

	tpl = template.Must(template.New("main").Funcs(template.FuncMap{
		"hextotext": hextotext,
	}).ParseFiles(
		"views/chain.html",
		"views/chains.html",
		"views/dblock.html",
		"views/eblock.html",
		"views/entrylist.html",
		"views/header.html",
		"views/index.html",
		"views/sentry.html",
	))

	server.Get(`/(?:home)?`, handleHome)
	server.Get(`/`, handleDBlocks)
	server.Get(`/index.html`, handleDBlocks)
	server.Get(`/chains/?`, handleChains)
	server.Get(`/chain/([^/]+)?`, handleChain)
	server.Get(`/dblocks/?`, handleDBlocks)
	server.Get(`/dblock/([^/]+)?`, handleDBlock)
	server.Get(`/eblock/([^/]+)?`, handleEBlock)
	server.Get(`/entry/([^/]+)?`, handleEntry)
	server.Get(`/sentry/([^/]+)?`, handleEntry)
	server.Post(`/search/?`, handleSearch)
	
	server.Run(fmt.Sprintf(":%d", cfg.PortNumber))
}

//func Start(dbref database.Db) {
//	db = dbref
//	ExtIDMap, _ = db.InitializeExternalIDMap() // reinitialized in restapi after a block is created
//	fmt.Println("explorer serving at port: 8087")
//	//http.ListenAndServe(":8087", nil)
//	go server.Run(":8087")
//
//}

func handleSearch(ctx *web.Context) {
	fmt.Println("r.Form:", ctx.Params["searchType"])
	fmt.Println("r.Form:", ctx.Params["searchText"])

//	pagesize := 1000
//	hashArray := make([]*notaryapi.Hash, 0, 5)
	searchText := strings.ToLower(strings.TrimSpace(ctx.Params["searchText"]))

	switch searchType := ctx.Params["searchType"]; searchType {
	case "entry":
		handleEntry(ctx, searchText)

	case "eblock":
		handleEBlock(ctx, searchText)

	case "dblock":
		handleDBlock(ctx, searchText)
	case "extID":
//		for key, _ := range ExtIDMap {
//			if strings.Contains(key[32:], searchText) {
//				hash := new(notaryapi.Hash)
//				hash.Bytes = []byte(key[:32])
//				hashArray = append(hashArray, hash)
//			}
//			if len(hashArray) > pagesize {
//				break
//			}
//		}
	default:
	}

//	tpl.ExecuteTemplate(ctx, "entrylist.html", hashArray)
}

func handleChain(ctx *web.Context, hash string) {
	chain, err := factom.GetChain(hash)
	if err != nil {
		fmt.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "chain.html", chain)
}

func handleChains(ctx *web.Context) {
	chains, err := factom.GetChains()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(chains)
	tpl.ExecuteTemplate(ctx, "chains.html", chains)
	fmt.Println("tpl.Execute finished")
}

func handleDBlocks(ctx *web.Context) {
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

	tpl.ExecuteTemplate(ctx, "index.html", dBlocks)
}

func handleDBlock(ctx *web.Context, hash string) {
	dblock, err := factom.GetDBlock(hash)
	if err != nil {
		log.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "dblock.html", dblock)
}

func handleEBlock(ctx *web.Context, mr string) {
	eblock, err := factom.GetEBlock(mr)
	if err != nil {
		log.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "eblock.html", eblock)
}

func handleEntry(ctx *web.Context, hash string) {
	entry, err := factom.GetEntry(hash)
	if err != nil {
		log.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "sentry.html", entry)
}

func handleHome(ctx *web.Context) {
	handleDBlocks(ctx)
}

func hextotext(h string) string {
	p, err := hex.DecodeString(h)
	if err != nil {
		log.Println(err)
	}
	return string(p)
}
