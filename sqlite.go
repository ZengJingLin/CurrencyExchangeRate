package main

import (
	"database/sql"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func (server *APIServer) InitializeSqlite(db *sql.DB) {
	//Create currency_info
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS currency_info (currency_id INTEGER PRIMARY KEY AUTOINCREMENT, currency_type TEXT, currency_price TEXT, create_datetime datetime, update_datetime datetime)")
	checkError("Initialize currency_info", err)
	stmt.Exec()

	//Create currency_info
	stmt, err = db.Prepare("CREATE TABLE IF NOT EXISTS currency_log (log_id INTEGER PRIMARY KEY AUTOINCREMENT, currency_type TEXT, orignal_price TEXT, new_price TEXT, create_datetime datetime)")
	checkError("Initialize currency_log", err)
	stmt.Exec()

}

func (server *APIServer) Insert(db *sql.DB, currency *Currency) {
	//Insert
	stmt, err := db.Prepare("INSERT INTO currency_info(currency_type, currency_price, create_datetime, update_datetime) VALUES(?,?,?,?)")
	checkError("Insert", err)

	datetime := time.Now()
	result, err := stmt.Exec(currency.CurrencyType, currency.CurrencyPrice, datetime, datetime)
	checkError("Insert result", err)

	id, err := result.LastInsertId()
	checkError("Insert id:"+strconv.FormatInt(id, 10), err)
}

func (server *APIServer) Update(db *sql.DB, currency *Currency) {
	var orignalPrice string

	datetime := time.Now()

	sentence := "SELECT currency_price FROM currency_info WHERE currency_type='" + currency.CurrencyType + "'"
	rows, err := db.Query(sentence)

	for rows.Next() {
		err = rows.Scan(&orignalPrice)
		checkError("Update rows scan", err)
	}

	stmt, err := db.Prepare("UPDATE currency_info SET currency_price=?, update_datetime=?  WHERE currency_type=?")
	checkError("Update", err)

	result, err := stmt.Exec(currency.CurrencyPrice, datetime, currency.CurrencyType)
	checkError("Update result", err)

	affect, err := result.RowsAffected()
	checkError("Update affect rows:"+strconv.FormatInt(affect, 10), err)

	server.recordLog(db, currency, orignalPrice)
}

func (server *APIServer) Select(db *sql.DB) *map[string]*Currency {
	rows, err := db.Query("SELECT currency_type,currency_price FROM currency_info")
	checkError("Select rows", err)
	defer rows.Close()
	currencyResult := make(map[string]*Currency)

	for rows.Next() {
		var currency_type string
		var currency_price string

		err = rows.Scan(&currency_type, &currency_price)
		checkError("Rows scan", err)

		currencyResult[currency_type] = &Currency{currency_type, currency_price}
	}

	return &currencyResult
}

func (server *APIServer) Delete(db *sql.DB, currency *Currency) {
	//Delete
	//stmt, err := db.Prepare(`DELETE FROM currency_info WHERE currency_type='ETH'`)
	stmt, err := db.Prepare("DELETE FROM currency_info WHERE currency_type=?")
	checkError("Delete", err)

	result, err := stmt.Exec(currency.CurrencyType)
	checkError("Delete result", err)

	affect, err := result.RowsAffected()
	checkError("Delete affect:"+strconv.FormatInt(affect, 10), err)
}

func (server *APIServer) recordLog(db *sql.DB, currency *Currency, orignalPrice string) {
	//Insert
	datetime := time.Now()
	stmt, err := db.Prepare("INSERT INTO currency_log(currency_type, orignal_price, new_price, create_datetime) Values(?,?,?,?)")
	checkError("recordLog", err)

	result, err := stmt.Exec(currency.CurrencyType, orignalPrice, currency.CurrencyPrice, datetime)
	checkError("recordLog result", err)

	id, err := result.LastInsertId()
	checkError("recordLog id:"+strconv.FormatInt(id, 10), err)
}
