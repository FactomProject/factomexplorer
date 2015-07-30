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
	"strconv"
	//"strings"

	"encoding/json"
	"github.com/FactomProject/factom"
	"github.com/hoisie/web"
)

var (
	cfg    = ReadConfig().Explorer
	server = web.NewServer()
	tpl    = new(template.Template)
)

func main() {
	var (
		err error
		dir string
	)

	server.Config.StaticDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.StaticDir != "" {
		server.Config.StaticDir = cfg.StaticDir
		dir = cfg.StaticDir
	}
	dir = "."

	tpl = template.Must(template.New("main").Funcs(template.FuncMap{
		"hashfilter":            hashfilter,
		"hextotext":             hextotext,
		"blockPrefixFilter":     blockPrefixFilter,
		"chainNamePrefixFilter": chainNamePrefixFilter,
	}).ParseFiles(
		dir+"/views/404.html",
		dir+"/views/chain.html",
		dir+"/views/chains.html",
		dir+"/views/cheader.html",
		dir+"/views/dblock.html",
		dir+"/views/eblock.html",
		dir+"/views/block.html",
		dir+"/views/entries.html",
		dir+"/views/header.html",
		dir+"/views/index.html",
		dir+"/views/pagination.html",
		dir+"/views/entry.html",
	))

	server.Get(`/(?:home)?`, handleHome)
	/*server.Get(`/`, handleDBlocks)
	server.Get(`/index.html`, handleDBlocks)
	//server.Get(`/chains/?`, handleChains)
	//server.Get(`/chain/([^/]+)?`, handleChain)
	server.Get(`/dblocks/?`, handleDBlocks)*/
	server.Get(`/dblock/([^/]+)?`, handleDBlock)
	server.Get(`/eblock/([^/]+)?`, handleBlock)
	server.Get(`/ablock/([^/]+)?`, handleBlock)
	server.Get(`/ecblock/([^/]+)?`, handleBlock)
	server.Get(`/fblock/([^/]+)?`, handleBlock)
	server.Get(`/entry/([^/]+)?`, handleEntry)
	server.Get(`/entry/([^/]+)?`, handleEntry)
	/*server.Post(`/search/?`, handleSearch)*/
	server.Get(`/test`, test)
	server.Get(`/.*`, handle404)

	err = Synchronize()
	if err != nil {
		panic(err)
	}

	server.Run(fmt.Sprintf(":%d", cfg.PortNumber))
}

func test(ctx *web.Context) {
	head, err := factom.GetChainHead("000000000000000000000000000000000000000000000000000000000000000a")
	log.Printf("test - %v, %v", head.ChainHead, err)
	/*body, err := factom.GetDBlock(head.KeyMR)
	str, _ := EncodeJSONString(body)
	log.Printf("test - %v, %v", str, err)
	Synchronize()*/
}

func EncodeJSONString(data interface{}) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(encoded), err
}

func handle404(ctx *web.Context) {
	var c interface{}
	tpl.ExecuteTemplate(ctx, "404.html", c)
}

/*
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
		handleEntryEid(ctx, searchText)
	default:
	}
}
*/
/*
func handleChain(ctx *web.Context, hash string) {
	chain, err := factom.GetChain(hash)
	if err != nil {
		log.Println(err)
		handle404(ctx)
		return
	}

	tpl.ExecuteTemplate(ctx, "chain.html", chain)
}*/
/*
func handleChains(ctx *web.Context) {
	chains, err := factom.GetChains()
	if err != nil {
		log.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "chains.html", chains)
}*/

func handleDBlock(ctx *web.Context, keyMR string) {
	Synchronize()
	type fullblock struct {
		DBlock DBlock
		DBInfo DBInfo
	}

	dblock, err := GetDBlock(keyMR)
	if err != nil {
		log.Println(err)
		handle404(ctx)
		return
	}
	dbinfo, err := GetDBInfo(keyMR)
	if err != nil {
		log.Println(err)
	}

	b := fullblock{
		DBlock: dblock,
		DBInfo: dbinfo,
	}

	tpl.ExecuteTemplate(ctx, "dblock.html", b)
}

func handleDBlocks(ctx *web.Context) {
	Synchronize()
	type dblockPlus struct {
		DBlocks  []DBlock
		PageInfo *PageState
	}

	height := GetBlockHeight()
	dBlocks := GetDBlocks(0, height)

	d := dblockPlus{
		DBlocks: dBlocks,
		PageInfo: &PageState{
			Current: 1,
			Max:     (len(dBlocks) / 50) + 1,
		},
	}

	page := 1
	var err error
	if p := ctx.Params["page"]; p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			log.Println(err)
			handle404(ctx)
			return
		}
		d.PageInfo.Current = page
	}
	if page > d.PageInfo.Max {
		handle404(ctx)
		return
	}
	if i, j := 50*(page-1), 50*page; len(dBlocks) > j {
		d.DBlocks = d.DBlocks[i:j]
	} else {
		d.DBlocks = d.DBlocks[i:]
	}

	tpl.ExecuteTemplate(ctx, "index.html", d)
}

func handleBlock(ctx *web.Context, mr string) {
	log.Printf("handleBlock - %v\n", mr)
	type blockPlus struct {
		Block    Block
		Hash     string
		Count    int
		PageInfo *PageState
	}

	block, err := GetBlock(mr)
	if err != nil {
		log.Printf("handleEBlock - factom.GetEBlock\n")
		log.Println(err)
		handle404(ctx)
		return
	}

	e := blockPlus{
		Block: block,
		Hash:  mr,
		Count: len(block.EntryList),
		PageInfo: &PageState{
			Current: 1,
			Max:     (len(block.EntryList) / 50) + 1,
		},
	}

	page := 1
	if p := ctx.Params["page"]; p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			log.Printf("handleEBlock - strconv\n")
			log.Println(err)
			handle404(ctx)
			return
		}
		e.PageInfo.Current = page
	}
	if page > e.PageInfo.Max {
		log.Printf("handleEBlock - e.PageInfo.Max\n")
		handle404(ctx)
		return
	}
	if i, j := 50*(page-1), 50*page; len(block.EntryList) > j {
		e.Block.EntryList = e.Block.EntryList[i:j]
	} else {
		e.Block.EntryList = e.Block.EntryList[i:]
	}

	tpl.ExecuteTemplate(ctx, "block.html", e)
}

func handleEntry(ctx *web.Context, hash string) {
	entry, err := GetEntry(hash)
	if err != nil {
		log.Println(err)
		handle404(ctx)
		return
	}

	tpl.ExecuteTemplate(ctx, "entry.html", entry)
}

/*
func handleEntryEid(ctx *web.Context, eid string) {
	entries, err := factom.GetEntriesByExtID(eid)
	if err != nil {
		log.Println(err)
		handle404(ctx)
		return
	}

	tpl.ExecuteTemplate(ctx, "entries.html", entries)
}*/

func handleHome(ctx *web.Context) {
	Synchronize()
	handleDBlocks(ctx)
}

type PageState struct {
	Current int
	Max     int
}

func (p *PageState) Next() int {
	return p.Current + 1
}

func (p *PageState) Next1() int {
	return p.Current + 2
}

func (p *PageState) Next2() int {
	return p.Current + 3
}

func (p *PageState) Prev() int {
	return p.Current - 1
}

func hextotext(h string) string {
	p, err := hex.DecodeString(h)
	if err != nil {
		log.Println(err)
	}
	return string(p)
}

func hashfilter(s string) string {
	var filter = []string{
		"0000000000000000000000000000000000000000000000000000000000000000",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}

	for _, v := range filter {
		if s == v {
			return "None"
		}
	}

	return s
}

func blockPrefixFilter(s string) string {
	switch s {
	case "000000000000000000000000000000000000000000000000000000000000000a":
		return "ablock"
	case "000000000000000000000000000000000000000000000000000000000000000c":
		return "ecblock"
	case "000000000000000000000000000000000000000000000000000000000000000f":
		return "fblock"
	}
	return "eblock"
}

func chainNamePrefixFilter(s string) string {
	switch s {
	case "000000000000000000000000000000000000000000000000000000000000000a":
		return "Admin"
	case "000000000000000000000000000000000000000000000000000000000000000c":
		return "Entry Credit"
	case "000000000000000000000000000000000000000000000000000000000000000f":
		return "Factoid"
	}
	return "Entry"
}
