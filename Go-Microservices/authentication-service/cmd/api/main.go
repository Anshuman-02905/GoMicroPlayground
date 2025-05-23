package main

import (
	"authentication/data"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = "80"

var counts int64

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting Authentication Service")
	//TODO CONNECT to DB
	conn := connectToDB()
	if conn == nil {
		log.Panic("Cannot Connect to Postgres")
	}

	//setup config

	app := Config{
		DB:     conn,
		Models: data.New(conn),
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic()
	}

}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func connectToDB() *sql.DB {
	dsn := os.Getenv("DSN")

	for {
		conection, err := openDB(dsn)
		if err != nil {
			log.Println("PostgresNotReady")
			counts++
		} else {
			log.Println("Connected to Postgres")
			return conection
		}
		if counts > 10 {
			log.Println(err)
			return nil
		}
		log.Println("Backing off for 2 seconds ...")
		time.Sleep(2 * time.Second)
		continue
	}
}
