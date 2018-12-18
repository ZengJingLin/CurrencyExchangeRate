package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var cfg APIConfig

type APIConfig struct {
	Listen               string `json:"listen"`
	StatsCollectInterval string `json:"statsCollectInterval"`
}

type APIServer struct {
	config     *APIConfig
	currency   map[string]*Currency
	currencyMu sync.RWMutex
	statsIntv  time.Duration
	db         *sql.DB
}

type Currency struct {
	CurrencyType  string
	CurrencyPrice string
}

func main() {
	readConfig(&cfg)
	startAPI()
}

func readConfig(cfg *APIConfig) {
	configFileName := "config.json"

	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}

	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)

	if err != nil {
		log.Fatal("File error: ", err.Error())
	}

	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)

	if err := jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func startAPI() {
	s := NewAPIServer(&cfg)
	s.Start()
}

func NewAPIServer(cfg *APIConfig) *APIServer {
	return &APIServer{
		config:   cfg,
		currency: make(map[string]*Currency),
	}
}

func (server *APIServer) Start() {
	var err error

	//檢查是否有Sqlite檔案
	fileExists, err := checkFileExists("./CurrencyExchangeRate.db")
	if fileExists != true {
		os.Create("./CurrencyExchangeRate.db")
	}

	//建立Sqlite連線
	db, err := sql.Open("sqlite3", "./CurrencyExchangeRate.db")
	server.db = db
	server.InitializeSqlite(server.db)

	checkError("Open Sqlite", err)
	//資料刷新間隔時間
	server.statsIntv, err = time.ParseDuration(server.config.StatsCollectInterval)

	statsTimer := time.NewTimer(server.statsIntv)
	log.Printf("Set currency data reflush interval to %v", server.statsIntv)

	//讀取資料庫中的貨幣資料
	server.getCurrencyData()

	//定時刷新資料
	go func() {
		for {
			select {
			case <-statsTimer.C:
				server.getCurrencyData()
				statsTimer.Reset(server.statsIntv)
			}
		}
	}()

	log.Printf("Starting API on %v", server.config.Listen)
	server.listen()
}

func (server *APIServer) listen() {
	route := mux.NewRouter()
	route.HandleFunc("/API/Insert/{currency}/{price}", server.insertCurrency)
	route.HandleFunc("/API/Select/{currency}", server.selectCurrency)
	route.HandleFunc("/API/Update/{currency}/{price}", server.updateCurrency)
	route.HandleFunc("/API/Delete/{currency}", server.deleteCurrency)
	route.NotFoundHandler = http.HandlerFunc(notFound)
	err := http.ListenAndServe(server.config.Listen, route)
	if err != nil {
		log.Fatalf("Failed to start API: %v", err)
	}
}

func (server *APIServer) getCurrencyData() {
	server.currency = *server.Select(server.db)
}

func (server *APIServer) insertCurrency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")

	currency := strings.ToUpper(mux.Vars(r)["currency"])
	price := strings.ToLower(mux.Vars(r)["price"])

	server.currencyMu.Lock()
	defer server.currencyMu.Unlock()

	if len(currency) != 0 || currency != "" {
		if isNumeric(price) {
			_, ok := server.currency[currency]

			if !ok {
				currencyData := &Currency{
					CurrencyType:  currency,
					CurrencyPrice: price,
				}

				channelInsert := make(chan *Currency, 200)
				channelInsertResult := make(chan bool, 200)

				channelInsert <- currencyData

				go func() {
					currencyDataInput := <-channelInsert
					server.Insert(server.db, currencyDataInput)
					server.getCurrencyData()
					channelInsertResult <- true
				}()

				<-channelInsertResult
				w.WriteHeader(http.StatusOK)
				reply := currency + " insert success"
				err := json.NewEncoder(w).Encode(reply)
				checkError("insertCurrency", err)
			} else {
				w.WriteHeader(http.StatusOK)
				reply := currency + " already exists"
				err := json.NewEncoder(w).Encode(reply)
				checkError("InsertCurrencyAlreadyExists", err)
			}

		} else {
			w.WriteHeader(http.StatusOK)
			reply := "Insert price is not number"
			err := json.NewEncoder(w).Encode(reply)
			checkError("InsertPriceIsNotNumber", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		reply := "Insert currency is null or empty"
		err := json.NewEncoder(w).Encode(reply)
		checkError("InsertCurrencyIsEmpty", err)
	}
}

func (server *APIServer) selectCurrency(w http.ResponseWriter, r *http.Request) {
	channelSelect := make(chan *http.Request, 200)
	channelSelectResult := make(chan bool, 200)

	channelSelect <- r

	go func() {
		currencyHTTPRequest := <-channelSelect
		currencyDataInput := strings.ToUpper(mux.Vars(currencyHTTPRequest)["currency"])

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")

		if len(currencyDataInput) != 0 || currencyDataInput != "" {
			reply, ok := server.currency[currencyDataInput]

			if !ok {
				w.WriteHeader(http.StatusOK)
				reply := "No " + currencyDataInput + " information"
				err := json.NewEncoder(w).Encode(reply)
				checkError("selectCurrencyNoResult", err)
			} else {
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(reply)
				checkError("selectCurrency", err)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			reply := "Select currency is null or empty"
			err := json.NewEncoder(w).Encode(reply)
			checkError("SelectCurrencyIsEmpty", err)
		}

		channelSelectResult <- true
	}()

	<-channelSelectResult
}

func (server *APIServer) updateCurrency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")

	currency := strings.ToUpper(mux.Vars(r)["currency"])
	price := strings.ToLower(mux.Vars(r)["price"])

	server.currencyMu.Lock()
	defer server.currencyMu.Unlock()

	if len(currency) != 0 || currency != "" {
		if isNumeric(price) {
			_, ok := server.currency[currency]

			if !ok {
				w.WriteHeader(http.StatusOK)
				reply := "No " + currency + " currency information for update"
				err := json.NewEncoder(w).Encode(reply)
				checkError("NoCurrencyForUpdate", err)
			} else {
				currencyData := &Currency{
					CurrencyType:  currency,
					CurrencyPrice: price,
				}

				channelUpdate := make(chan *Currency, 200)
				channelUpdateResult := make(chan bool, 200)

				channelUpdate <- currencyData

				go func() {
					currencyDataInput := <-channelUpdate
					server.Update(server.db, currencyDataInput)
					server.getCurrencyData()
					channelUpdateResult <- true
				}()

				<-channelUpdateResult
				w.WriteHeader(http.StatusOK)
				reply := "Update " + currency + " success"
				err := json.NewEncoder(w).Encode(reply)
				checkError("updateCurrency", err)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			reply := "Update price is not number"
			err := json.NewEncoder(w).Encode(reply)
			checkError("UpdatePriceIsNotNumber", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		reply := "Update currency is null or empty"
		err := json.NewEncoder(w).Encode(reply)
		checkError("UpdateCurrencyIsEmpty", err)
	}

}

func (server *APIServer) deleteCurrency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")

	currency := strings.ToUpper(mux.Vars(r)["currency"])
	price := strings.ToLower(mux.Vars(r)["price"])

	server.currencyMu.Lock()
	defer server.currencyMu.Unlock()

	_, ok := server.currency[currency]

	if !ok {
		w.WriteHeader(http.StatusOK)
		reply := "No " + currency + " currency information for delete"
		err := json.NewEncoder(w).Encode(reply)
		checkError("NoCurrencyForUpdate", err)
	} else {
		currencyData := &Currency{
			CurrencyType:  currency,
			CurrencyPrice: price,
		}

		channelDelete := make(chan *Currency, 200)
		channelDeleteResult := make(chan bool, 200)
		channelDelete <- currencyData

		go func() {
			currencyDataInput := <-channelDelete
			server.Delete(server.db, currencyDataInput)
			server.getCurrencyData()
			channelDeleteResult <- true
		}()

		<-channelDeleteResult
		w.WriteHeader(http.StatusOK)
		reply := "Delete " + currency + " success"
		err := json.NewEncoder(w).Encode(reply)
		checkError("deleteCurrency", err)
	}
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
}

func checkError(funcName string, err error) {
	if err != nil {
		log.Printf("%v error: %v", funcName, err)
		return
	}
}

func checkFileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func isNumeric(inputString string) bool {
	_, err := strconv.ParseFloat(inputString, 64)
	return err == nil
}
