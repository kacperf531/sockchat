package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/storage"
)

func main() {

	godotenv.Load("../.env")
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@tcp("+os.Getenv("DB_HOST")+")/sockchat")
	if err != nil {
		log.Fatalf("could not connect to the DB due to an error: %v", err)
	}
	store, err := sockchat.NewSockChatStore()
	users := storage.NewUserStore(db)

	if err != nil {
		log.Fatalf("problem creating file system player store, %v ", err)
	}

	server := sockchat.NewSockChatServer(store, users)

	log.Fatal(http.ListenAndServe(":5000", server))
}
