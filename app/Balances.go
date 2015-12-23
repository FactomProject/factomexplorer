package app

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ThePiachu/Go/Log"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	Response string
	Success  bool
}

//var server string = "localhost:8088/" //Localhost
//var server string = "52.17.183.121:8088/" //TestNet
var server string = "52.18.72.212:8088/" //MainNet

func FactomdFactoidBalance(c appengine.Context, adr string) (int64, error) {
	resp, err := Call(c, fmt.Sprintf("http://%s/v1/factoid-balance/%s", server, adr))
	if err != nil {
		Log.Errorf(c, "FactomdFactoidBalance - %v", err)
		return 0, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf(c, "FactomdFactoidBalance - %v", err)
		return 0, err
	}
	resp.Body.Close()

	b := new(Response)
	if err := json.Unmarshal(body, b); err != nil {
		Log.Errorf(c, "FactomdFactoidBalance - %v", err)
		return 0, err
	}

	if !b.Success {
		return 0, fmt.Errorf("%s", b.Response)
	}

	v, err := strconv.ParseInt(b.Response, 10, 64)
	if err != nil {
		Log.Errorf(c, "FactomdFactoidBalance - %v", err)
		return 0, err
	}

	return v, nil

}

func FactomdECBalance(c appengine.Context, adr string) (int64, error) {
	resp, err := Call(c, fmt.Sprintf("http://%s/v1/entry-credit-balance/%s", server, adr))
	if err != nil {
		Log.Errorf(c, "FactomdECBalance - %v", err)
		return 0, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf(c, "FactomdECBalance - %v", err)
		return 0, err
	}
	resp.Body.Close()

	b := new(Response)
	if err := json.Unmarshal(body, b); err != nil {
		Log.Errorf(c, "FactomdECBalance - %v", err)
		return 0, err
	}

	if !b.Success {
		return 0, fmt.Errorf("%s", b.Response)
	}

	v, err := strconv.ParseInt(b.Response, 10, 64)
	if err != nil {
		Log.Errorf(c, "FactomdECBalance - %v", err)
		return 0, err
	}

	return v, nil
}

type Data struct {
	Data string
}

func FactomdGetRaw(c appengine.Context, keymr string) ([]byte, error) {
	resp, err := Call(c, fmt.Sprintf("http://%s/v1/get-raw-data/%s", server, keymr))
	if err != nil {
		Log.Errorf(c, "FactomdGetRaw - %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf(c, "FactomdGetRaw - %v", err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(string(body))
	}

	d := new(Data)
	if err := json.Unmarshal(body, d); err != nil {
		Log.Errorf(c, "FactomdGetRaw - %v", err)
		return nil, err
	}

	raw, err := hex.DecodeString(d.Data)
	if err != nil {
		Log.Errorf(c, "FactomdGetRaw - %v", err)
		return nil, err
	}

	return raw, nil
}

type FactomdDBlock struct {
	DBHash string
	Header struct {
		PrevBlockKeyMR string
		Timestamp      uint64
		SequenceNumber int
	}
	EntryBlockList []struct {
		ChainID string
		KeyMR   string
	}
}

type FactomdDBlockHead struct {
	KeyMR string
}

func FactomdGetDBlock(c appengine.Context, keymr string) (*FactomdDBlock, error) {
	resp, err := Call(c, fmt.Sprintf("http://%s/v1/directory-block-by-keymr/%s", server, keymr))
	if err != nil {
		Log.Errorf(c, "FactomdGetDBlock - %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf(c, "FactomdGetDBlock - %v", err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(string(body))
	}

	d := new(FactomdDBlock)
	if err := json.Unmarshal(body, d); err != nil {
		return nil, fmt.Errorf("%s: %s\n", err, body)
	}

	return d, nil
}

func FactomdGetDBlockHead(c appengine.Context) (*FactomdDBlockHead, error) {
	resp, err := Call(c, fmt.Sprintf("http://%s/v1/directory-block-head/", server))
	if err != nil {
		Log.Errorf(c, "FactomdGetDBlockHead - %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf(c, "FactomdGetDBlockHead - %v", err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(string(body))
	}

	d := new(FactomdDBlockHead)
	json.Unmarshal(body, d)

	/*d := new(FactomdDBlockHead)
	d.KeyMR = "9c83bab565fd625f1f575dd8fe6545b354c970e29ce31dbe5f75470b717d168d"*/
	return d, nil
}

func Call(c appengine.Context, url string) (*http.Response, error) {
	TimeoutDuration, err := time.ParseDuration("60s")
	if err != nil {
		Log.Errorf(c, "CallJSON - %v", err)
		return nil, err
	}
	tr := urlfetch.Transport{Context: c, Deadline: TimeoutDuration}
	client := http.Client{Transport: &tr}

	//sending request
	resp, err := client.Get(url)
	return resp, err
}
