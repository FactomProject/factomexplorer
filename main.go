package main

import (
	//"flag"
	"fmt"
	"github.com/FactomProject/dynrsrc"
	"github.com/FactomProject/gobundle"
	"os"
	"log"
	"code.google.com/p/gcfg"
	"github.com/FactomProject/FactomCode/database"	
	"github.com/FactomProject/FactomCode/database/ldb"		
	"github.com/FactomProject/FactomCode/factomapi"	
	"time"		
)

//var portNumber = flag.Int("p", 8087, "Set the port to listen on")
var (
 	logLevel = "DEBUG"
	portNumber int = 8087  	
	applicationName = "factom/client"
	serverAddr = "localhost:8083"	
	ldbpath = "/tmp/client/ldb9"	
	dataStorePath = "/tmp/client/seed/csv"
	refreshInSeconds int = 60
	pagesize = 1000 	
	
	db database.Db // database
	
	//Map to store imported csv files
	clientDataFileMap map[string]string	
	extIDMap map[string]bool	
	
)
func watchError(err error) {
	panic(err)
}

func readError(err error) {
	fmt.Println("error: ", err)
}

func init() {
	
	loadConfigurations()
	factomapi.SetServerAddr(serverAddr)
	
	initDB()
		
	gobundle.Setup.Application.Name = applicationName
	gobundle.Init()
	
	err := dynrsrc.Start(watchError, readError)
	if err != nil { panic(err) }
	
	loadStore()
	loadSettings()
	templates_init()
	serve_init()
	extIDMap, _ = db.InitializeExternalIDMap()

	// Import data related to new factom blocks created on server
	ticker := time.NewTicker(time.Second * time.Duration(refreshInSeconds)) 
	go func() {
		for _ = range ticker.C {
			//downloadAndImportDbRecords()
			RefreshEntries()
		}
	}()		
}

func main() {
	defer func() {
		dynrsrc.Stop()
		server.Close()
	}()
	
	server.Run(fmt.Sprint(":", portNumber))
	
}

func loadConfigurations(){
	cfg := struct {
		App struct{
			PortNumber	int		
			ApplicationName string
			ServerAddr string
			DataStorePath string
			RefreshInSeconds int
			PageSize int
	    }
		Log struct{
	    	LogLevel string
		}
    }{}
	
	wd, err := os.Getwd()
	if err != nil{
		log.Println(err)
	}	
	err = gcfg.ReadFileInto(&cfg, wd+"/client.conf")
	if err != nil{
		log.Println(err)
		log.Println("Client starting with default settings...")
	} else {
	
		//setting the variables by the valued form the config file
		logLevel = cfg.Log.LogLevel	
		applicationName = cfg.App.ApplicationName
		portNumber = cfg.App.PortNumber
		serverAddr = cfg.App.ServerAddr
		dataStorePath = cfg.App.DataStorePath
		refreshInSeconds = cfg.App.RefreshInSeconds
		pagesize = cfg.App.PageSize
	}
	
}

func initDB() {
	
	//init db
	var err error
	db, err = ldb.OpenLevelDB(ldbpath, false)
	
	if err != nil{
		log.Println("err opening db: %v", err)
	}
	
	if db == nil{
		log.Println("Creating new db ...")			
		db, err = ldb.OpenLevelDB(ldbpath, true)
		
		if err!=nil{
			panic(err)
		}		
	}
	
	log.Println("Database started from: " + ldbpath)

}