// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package app

import (
	"appengine"
	"encoding/hex"
	"encoding/json"
	"github.com/ThePiachu/Go/Log"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	tpl = new(template.Template)
)
var blocksPerPAge int = 50

func init() {
	tpl = template.Must(template.New("main").Funcs(template.FuncMap{
		"hashfilter":            hashfilter,
		"hextotext":             hextotext,
		"blockPrefixFilter":     blockPrefixFilter,
		"chainNamePrefixFilter": chainNamePrefixFilter,
	}).ParseFiles(
		"views/404.html",
		"views/chain.html",
		"views/chains.html",
		"views/cheader.html",
		"views/dblock.html",
		"views/eblock.html",
		"views/block.html",
		"views/entries.html",
		"views/header.html",
		"views/index.html",
		"views/pagination.html",
		"views/entry.html",
		"views/address.html",
	))

	http.HandleFunc(`/Admin/upkeep`, Upkeep)
	http.HandleFunc(`/chains/`, handleChains)
	http.HandleFunc(`/chain/`, handleChain)
	http.HandleFunc(`/dblocks/`, handleDBlocks)
	http.HandleFunc(`/dblock/`, handleDBlock)
	http.HandleFunc(`/block/`, handleBlock)
	http.HandleFunc(`/eblock/`, handleBlock)
	http.HandleFunc(`/ablock/`, handleBlock)
	http.HandleFunc(`/ecblock/`, handleBlock)
	http.HandleFunc(`/fblock/`, handleBlock)
	http.HandleFunc(`/entry/`, handleEntry)
	http.HandleFunc(`/address/`, handleAddress)
	http.HandleFunc(`/search`, handleSearch)
	http.HandleFunc(`/search/`, handleSearch)
	http.HandleFunc(`/test`, test)
	http.HandleFunc(`/index.html`, handleDBlocks)
	http.HandleFunc(`/.*`, handle404)
	http.HandleFunc(`/`, handleHome)

}

func Upkeep(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	SynchronizationGoroutine(c)
}

func getIndexParameter(r *http.Request) string {
	c := appengine.NewContext(r)
	searchText := strings.TrimSpace(r.FormValue("searchText"))
	if searchText != "" {
		Log.Debugf(c, "SearchText - %v", searchText)
		return searchText
	}
	params := strings.Split(r.URL.String(), "/")
	Log.Debugf(c, "params[len(params)-1] - %v", params[len(params)-1])
	return params[len(params)-1]
}

func test(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	c.Debugf("Test")
}

func EncodeJSONString(data interface{}) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(encoded), err
}

func handle404(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "404.html", nil)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	Log.Debugf(c, "handleSearch - `%v`, `%v`", r.FormValue("searchType"), r.FormValue("searchText"))

	//	pagesize := 1000
	//	hashArray := make([]*notaryapi.Hash, 0, 5)

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
}

func handleAddress(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	hash := getIndexParameter(r)
	address, err := GetAddressInformationFromFactom(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	tpl.ExecuteTemplate(w, "address.html", address)
}

func handleChain(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	hash := getIndexParameter(r)
	chain, err := GetChainByName(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	tpl.ExecuteTemplate(w, "chain.html", chain)
}

func handleChains(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	chains, err := GetChains(c)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	tpl.ExecuteTemplate(w, "chains.html", chains)
}

func handleDBlock(w http.ResponseWriter, r *http.Request) {
	keyMR := getIndexParameter(r)
	c := appengine.NewContext(r)
	type fullblock struct {
		DBlock *DBlock
		DBInfo DBInfo
	}

	dblock, err := GetDBlock(c, keyMR)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	b := fullblock{
		DBlock: dblock,
	}

	tpl.ExecuteTemplate(w, "dblock.html", b)
}

func handleDBlocks(w http.ResponseWriter, r *http.Request) {
	type dblockPlus struct {
		DBlocks  []*DBlock
		PageInfo *PageState
	}

	c := appengine.NewContext(r)
	Log.Debugf(c, "handleDBlocks")
	height := GetBlockHeight(c)
	dBlocks, err := GetDBlocksReverseOrder(c, 0, height)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	page := 1
	maxPage := (len(dBlocks) / blocksPerPAge) + 1
	if p := r.FormValue("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			Log.Errorf(c, "Error - %v", err)
			handle404(w, r)
			return
		}
	}

	if page > maxPage {
		handle404(w, r)
		return
	}
	d := dblockPlus{
		DBlocks:  dBlocks,
		PageInfo: NewPageState(page, maxPage),
	}
	if i, j := blocksPerPAge*(page-1), blocksPerPAge*page; len(dBlocks) > j {
		d.DBlocks = d.DBlocks[i:j]
	} else {
		d.DBlocks = d.DBlocks[i:]
	}

	tpl.ExecuteTemplate(w, "index.html", d)
}

func handleBlock(w http.ResponseWriter, r *http.Request) {
	mr := getIndexParameter(r)
	log.Printf("handleBlock - %v\n", mr)
	type blockPlus struct {
		Block    *Block
		Hash     string
		Count    int
		PageInfo *PageState
	}
	c := appengine.NewContext(r)

	block, err := GetBlock(c, mr)
	if err != nil {
		log.Printf("handleEBlock - factom.GetEBlock\n")
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	page := 1
	maxPage := (len(block.EntryList) / blocksPerPAge) + 1
	if p := r.FormValue("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			log.Printf("handleEBlock - strconv\n")
			Log.Errorf(c, "Error - %v", err)
			handle404(w, r)
			return
		}
	}
	if page > maxPage {
		log.Printf("handleEBlock - e.PageInfo.Max\n")
		handle404(w, r)
		return
	}
	e := blockPlus{
		Block:    block,
		Hash:     mr,
		Count:    len(block.EntryList),
		PageInfo: NewPageState(page, maxPage),
	}
	if i, j := blocksPerPAge*(page-1), blocksPerPAge*page; len(block.EntryList) > j {
		e.Block.EntryList = e.Block.EntryList[i:j]
	} else {
		e.Block.EntryList = e.Block.EntryList[i:]
	}

	tpl.ExecuteTemplate(w, "block.html", e)
}

func handleEntry(w http.ResponseWriter, r *http.Request) {
	hash := getIndexParameter(r)
	c := appengine.NewContext(r)
	entry, err := GetEntry(c, hash)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}

	tpl.ExecuteTemplate(w, "entry.html", entry)
}

func handleEntryEid(w http.ResponseWriter, r *http.Request) {
	eid := getIndexParameter(r)
	c := appengine.NewContext(r)
	Log.Debugf(c, "handleEntryEid - %v", eid)
	entries, err := GetEntriesByExtID(c, eid)
	if err != nil {
		Log.Errorf(c, "Error - %v", err)
		handle404(w, r)
		return
	}
	tpl.ExecuteTemplate(w, "entries.html", entries)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	Log.Debugf(c, "handleHome")
	handleDBlocks(w, r)
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
