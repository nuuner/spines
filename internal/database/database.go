package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Connect(path string) error {
	var err error
	DB, err = sql.Open("sqlite3", path+"?_foreign_keys=on&_loc=auto")
	if err != nil {
		return err
	}
	return DB.Ping()
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
