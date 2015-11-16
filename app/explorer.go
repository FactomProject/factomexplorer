// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"os"
	"github.com/hoisie/web"
)

var (
	tpl = new(template.Template)
	cfg    = ReadConfig().Explorer
	wserver = web.NewServer()
)
var blocksPerPage int = 10

func main() {
	var (
		err error
		dir string
	)
    OriginalInit()
    wserver.Config.StaticDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.StaticDir != "" {
		wserver.Config.StaticDir = cfg.StaticDir
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
		dir+"/views/address.html",
	))

	wserver.Get(`/(?:home)?`, handleHome)
	wserver.Get(`/`, handleDBlocks)
	wserver.Get(`/index.html`, handleDBlocks)
	wserver.Get(`/chains/?`, handleChains)
	wserver.Get(`/chain/([^/]+)?`, handleChain)
	wserver.Get(`/dblocks/?`, handleDBlocks)
	wserver.Get(`/dblock/([^/]+)?`, handleDBlock)
	wserver.Get(`/eblock/([^/]+)?`, handleBlock)
	wserver.Get(`/ablock/([^/]+)?`, handleBlock)
	wserver.Get(`/ecblock/([^/]+)?`, handleBlock)
	wserver.Get(`/fblock/([^/]+)?`, handleBlock)
	wserver.Get(`/entry/([^/]+)?`, handleEntry)
	wserver.Get(`/entry/([^/]+)?`, handleEntry)
	wserver.Get(`/address/([^/]+)?`, handleAddress)
	wserver.Post(`/search/?`, handleSearch)
	wserver.Get(`/test`, test)
	wserver.Get(`/.*`, handle404)

	go SynchronizationGoroutine()

	wserver.Run(fmt.Sprintf(":%d", cfg.PortNumber))
}

func Upkeep(ctx *web.Context) {
	SynchronizationGoroutine()
}

func getIndexParameter(r *http.Request) string {
	searchText := strings.TrimSpace(r.FormValue("searchText"))
	if searchText != "" {
		fmt.Println("SearchText - %v", searchText)
		return searchText
	}
	params := strings.Split(r.URL.String(), "/")
	params = strings.Split(params[len(params)-1], "?")

	return params[0]
}

func test(ctx *web.Context) {
	exIDs, err := ListExternalIDs()
	fmt.Println("test - %v, %v", exIDs, err)
}

func EncodeJSONString(data interface{}) (string, error) {
	encoded, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return "", err
	}
	return string(encoded), err
}

func handle404(ctx *web.Context) {
	var c interface{}
	tpl.ExecuteTemplate(ctx, "404.html", c)
}

//func handleSearch(w http.ResponseWriter, r *http.Request) {
func handleSearch(ctx *web.Context) {
	fmt.Println("r.Form:", ctx.Params["searchType"])
	fmt.Println("r.Form:", ctx.Params["searchText"])

	//	pagesize := 1000
	//	hashArray := make([]*notaryapi.Hash, 0, 5)
/*
	switch searchType := r.FormValue("searchType"); searchType {
	case "entry":
		handleEntry(w, r)
	case "block":
		handleBlock(w, r)
	case "dblock":
		handleDBlock(w, r)
	case "address":
		handleAddress(w, r)
	case "extID":
		handleEntryEid(w, r)
	default:
		handle404(w, r)
	}
}*/
	//	pagesize := 1000
	//	hashArray := make([]*notaryapi.Hash, 0, 5)
	searchText := strings.TrimSpace(ctx.Params["searchText"])

	switch searchType := ctx.Params["searchType"]; searchType {
	case "entry":
		handleEntry(ctx, searchText)
	case "eblock":
		handleBlock(ctx, searchText)
	case "block":
		handleBlock(ctx, searchText)
	case "dblock":
		handleDBlock(ctx, searchText)
	case "address":
		handleAddress(ctx, searchText)
		/*	case "extID":
			handleEntryEid(ctx, searchText)*/
	default:
		handle404(ctx)
	}
}

func handleAddress(ctx *web.Context, hash string) {
	address, err := GetAddressInformationFromFactom(hash)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		//handle404(w, r)
		handle404(ctx)
		return
	}

	tpl.ExecuteTemplate(ctx, "address.html", address)
}

//func handleChain(w http.ResponseWriter, r *http.Request) {
func handleChain(ctx *web.Context, hash string) {
	//hash := getIndexParameter(r)

	page := 1
	if p := ctx.Request.FormValue("page"); p!= "" {
		var err error
		page, err = strconv.Atoi(p)
		if err != nil {
			fmt.Errorf("Error - %v", err)
		    handle404(ctx)
			return
		}
	}

	min := (page - 1) * blocksPerPage
	if min < 0 {
		min = 0
	}

	chain, err := GetChainByName(hash, min, blocksPerPage)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		handle404(ctx)
		return
	}

	type chainPlus struct {
		Chain    *Chain
		PageInfo *PageState
	}

	pi := NewPageState(page, len(chain.Entries))
	if len(chain.Entries) == blocksPerPage {
		pi.Next = page + 1
	}
	if page > 0 {
		pi.Previous = page - 1
	}

	cp := chainPlus{Chain: chain, PageInfo: pi}

	tpl.ExecuteTemplate(ctx, "chain.html", cp)
}

func handleChains(ctx *web.Context) {
	chains, err := GetChains()
	if err != nil {
		fmt.Errorf("Error - %v", err)
		//handle404(w, r)
		handle404(ctx)
		return
	}

	tpl.ExecuteTemplate(ctx, "chains.html", chains)
}

func handleDBlock(ctx *web.Context, keyMR string) {
	type fullblock struct {
		DBlock *DBlock
		DBInfo DBInfo
	}

	dblock, err := GetDBlock(keyMR)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		handle404(ctx)
		return
	}

	b := fullblock{
		DBlock: dblock,
	}

	tpl.ExecuteTemplate(ctx, "dblock.html", b)
}

func handleDBlocks(ctx *web.Context) {
	type dblockPlus struct {
		DBlocks  []*DBlock
		PageInfo *PageState
	}

	fmt.Println("handleDBlocks")
	height := GetBlockHeight()

	page := 1
	maxPage := (height / blocksPerPage) + 1
	if p := ctx.Request.FormValue("page"); p!= "" {
		var err error
		page, err = strconv.Atoi(p)
		if err != nil {
			fmt.Errorf("Error - %v", err)
			handle404(ctx)
			return
		}
	}

	if page > maxPage {
		handle404(ctx)
		return
	}
	max := height - (page-1)*blocksPerPage
	min := max - blocksPerPage
	if min < 0 {
		min = 0
	}
	dBlocks, err := GetDBlocksReverseOrder(min, max)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		handle404(ctx)
		return
	}

	d := dblockPlus{
		DBlocks:  dBlocks,
		PageInfo: NewPageState(page, maxPage),
	}

	tpl.ExecuteTemplate(ctx, "index.html", d)
}

func handleBlock(ctx *web.Context, mr string) {
	log.Printf("handleBlock - %v\n", mr)
	type blockPlus struct {
		Block    *Block
		Hash     string
		Count    int
		PageInfo *PageState
	}

	block, err := GetBlock(mr)
	if err != nil {
		log.Printf("handleEBlock - factom.GetEBlock\n")
		fmt.Errorf("Error - %v", err)
		handle404(ctx)
		return
	}

	page := 1
	maxPage := (len(block.EntryList) / blocksPerPage) + 1
	if p := ctx.Request.FormValue("page"); p!= "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			log.Printf("handleEBlock - strconv\n")
			fmt.Errorf("Error - %v", err)
			handle404(ctx)
			return
		}
	}
	if page > maxPage {
		log.Printf("handleEBlock - e.PageInfo.Max\n")
		handle404(ctx)
		return
	}
	e := blockPlus{
		Block:    block,
		Hash:     mr,
		Count:    len(block.EntryList),
		PageInfo: NewPageState(page, maxPage),
	}
	if i, j := blocksPerPage*(page-1), blocksPerPage*page; len(block.EntryList) > j {
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

func handleEntryEid(ctx *web.Context, eid string) {
	fmt.Println("handleEntryEid - %v", eid)
	entries, err := GetEntriesByExtID(eid)
	if err != nil {
		fmt.Errorf("Error - %v", err)
		handle404(ctx)
		return
	}
	tpl.ExecuteTemplate(ctx, "entries.html", entries)
}

func handleHome(ctx *web.Context) {
	handleDBlocks(ctx)
}

type PageState struct {
	Current  int
	Max      int
	Previous int
	Next     int

	//For display purposes:
	// <LeftMostBatch ... LeftBatch Current RightBatch ... RightmostBatch >
	LeftmostBatch  []int
	LeftBatch      []int
	RightBatch     []int
	RightmostBatch []int
}

func NewPageState(current int, max int) *PageState {
	ps := new(PageState)
	ps.Current = current
	ps.Max = max

	ps.Previous = current - 1
	ps.Next = current + 1

	if current > 6 {
		ps.LeftmostBatch = []int{1, 2}
		ps.LeftBatch = []int{current - 2, current - 1}
	} else {
		ps.LeftBatch = make([]int, current-1)
		for i := 1; i < current; i++ {
			ps.LeftBatch[i-1] = i
		}
	}

	if current < max-5 {
		ps.RightBatch = []int{current + 1, current + 2}
		ps.RightmostBatch = []int{max - 1, max}
	} else {
		ps.RightBatch = make([]int, max-current)
		for i := 0; i < max-current; i++ {
			ps.RightBatch[i] = current + i + 1
		}
	}
	return ps
}

func hextotext(h string) string {
	p, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
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
	case AdminBlockID:
		return "ablock"
	case ECBlockID:
		return "ecblock"
	case FactoidBlockID:
		return "fblock"
	}
	return "eblock"
}

func chainNamePrefixFilter(s string) string {
	switch s {
	case AdminBlockID:
		return "Admin"
	case ECBlockID:
		return "Entry Credit"
	case FactoidBlockID:
		return "Factoid"
	}
	return "Entry"
}
