package factomexplorer

import (
	"fmt"
	"html/template"
	"log"
//	"net/http"

	"github.com/FactomProject/factom"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/hoisie/web"	
)

var _ = fmt.Sprint("tmp")

var (
	//handleStatic = http.FileServer(http.Dir("./"))
	tpl = new(template.Template)
	db              database.Db	
	server = web.NewServer()	
	
)

func init() {
	tpl = template.Must(template.ParseFiles(
		"views/index.html",
		"views/dblock.html",
		"views/eblock.html",
		"views/sentry.html",
	))
	server.Config.StaticDir	= "/tmp/static"
	server.Get(`/(?:home)?`, handleHome)	
	server.Get(`/`, handleDBlocks)
	server.Get(`/dblocks/`, handleDBlocks)
	server.Get(`/dblock/`, handleDBlock)
	server.Get(`/eblock/`, handleEBlock)
	server.Get(`/sentry/`, handleEntry)
	server.Post(`/search/`, handleSearch)	
}

func Start(dbref database.Db) {
	db = dbref
	fmt.Println("explorer serving at port: 8087")	
	//http.ListenAndServe(":8087", nil)
	go server.Run("localhost:8087")	

}


func handleSearch(ctx *web.Context) {

	fmt.Println("r.Form:", ctx.Params["searchText"])	

	//tpl.ExecuteTemplate(w, "index.html", dBlocks)
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

	tpl.ExecuteTemplate(ctx.ResponseWriter, "index.html", dBlocks)
}

func handleDBlock(ctx *web.Context) {
	mr := ctx.Request.URL.Path[len("/dblock/"):]
	
	dblock, err := factom.GetDBlock(mr)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(ctx.ResponseWriter, "dblock.html", dblock)
}

func handleEBlock(ctx *web.Context) {
	mr := ctx.Request.URL.Path[len("/eblock/"):]
	
	eblock, err := factom.GetEBlock(mr)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(ctx.ResponseWriter, "eblock.html", eblock)
}

func handleEntry(ctx *web.Context) {
	hash := ctx.Request.URL.Path[len("/entry/"):]
	
	entry, err := factom.GetEntry(hash)	
	if err != nil {
		fmt.Println(err)
	}
	
	tpl.ExecuteTemplate(ctx.ResponseWriter, "sentry.html", entry)
}

func handleHome(ctx *web.Context) {
	handleDBlocks(ctx)
}