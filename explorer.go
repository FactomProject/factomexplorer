package factomexplorer

import (
	"fmt"
	"html/template"
	"log"
	"strings"	

	"github.com/FactomProject/factom"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/notaryapi"		
	"github.com/hoisie/web"	
)

var _ = fmt.Sprint("tmp")

var (
	//handleStatic = http.FileServer(http.Dir("./"))
	tpl = new(template.Template)
	db              database.Db	
	server = web.NewServer()	
		
	ExtIDMap map[string]bool	
)

func init() {
	tpl = template.Must(template.ParseFiles(
		"views/dblock.html",
		"views/eblock.html",
		"views/entrylist.html",
		"views/header.html",
		"views/index.html",
		"views/sentry.html",
	))
	server.Config.StaticDir	= "/home/mjb/work/factom/go/src/github.com/FactomProject/factomexplorer"
	server.Get(`/(?:home)?`, handleHome)	
	server.Get(`/`, handleDBlocks)
	server.Get(`/index.html`, handleDBlocks)
	server.Get(`/dblocks/?`, handleDBlocks)
	server.Get(`/dblock/([^/]+)?`, handleDBlock)
	server.Get(`/eblock/([^/]+)?`, handleEBlock)
	server.Get(`/entry/([^/]+)?`, handleEntry)
	server.Post(`/search/?`, handleSearch)		

}

func Start(dbref database.Db) {
	db = dbref
	ExtIDMap, _ = db.InitializeExternalIDMap() // reinitialized in restapi after a block is created	
	fmt.Println("explorer serving at port: 8087")	
	//http.ListenAndServe(":8087", nil)
	go server.Run("localhost:8087")	

}


func handleSearch(ctx *web.Context) {

	fmt.Println("r.Form:", ctx.Params["searchType"])	
	fmt.Println("r.Form:", ctx.Params["searchText"])
	
	pagesize := 1000
	hashArray := make([]*notaryapi.Hash, 0, 5)
	searchText := ctx.Params["searchText"]
	searchText = strings.ToLower(strings.TrimSpace(searchText))	
	
	switch searchType := ctx.Params["searchType"]; searchType {
//	case "entry":
//		handleEntry(ctx)
//
//	case "eblock":
//		handleEBlock(ctx)
//			
//	case "dblock":
//		handleDBlock(ctx)
		
	case "extID":
		for key, _ := range ExtIDMap {
			if strings.Contains(key[32:], searchText){
				hash := new (notaryapi.Hash)
				hash.Bytes = []byte(key[:32])
				hashArray = append(hashArray, hash)
			}
			if len(hashArray) > pagesize {
				break
			}
		}		
		
	default:	

	}
		
	tpl.ExecuteTemplate(ctx, "entrylist.html", hashArray)
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

func handleDBlock(ctx *web.Context, mr string) {
	dblock, err := factom.GetDBlock(mr)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(ctx, "dblock.html", dblock)
}

func handleEBlock(ctx *web.Context, mr string) {
	eblock, err := factom.GetEBlock(mr)
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(ctx, "eblock.html", eblock)
}

func handleEntry(ctx *web.Context, hash string) {
	entry, err := factom.GetEntry(hash)	
	if err != nil {
		fmt.Println(err)
	}

	tpl.ExecuteTemplate(ctx, "sentry.html", entry)
}

func handleHome(ctx *web.Context) {
	handleDBlocks(ctx)
}