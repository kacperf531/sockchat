package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/storage"
)

const messagesIndex = "messages"

func main() {
	sockchat.TestRedisConnection()
	sockchat.TestElasticSearchConnection()

	godotenv.Load("../.env")
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@tcp("+os.Getenv("DB_HOST")+")/sockchat")
	if err != nil {
		log.Fatalf("could not create db object due to an error: %v", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("could not connect to the db due to an error: %v", err)
	}
	if os.Getenv("ENVIRONMENT") == "DEV" {
		err = storage.ResetUsersTable(db)
		if err != nil {
			log.Fatalf("error setting up the users table %v", err)
		}
	}

	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	messageStore := storage.NewMessageStore(es, messagesIndex)

	channelStore, err := sockchat.NewChannelStore(messageStore)
	users := storage.NewUserStore(db)

	server := sockchat.NewSockChatServer(channelStore, users, messageStore)

	if err != nil {
		log.Fatalf("problem creating file system player store, %v ", err)
	}

	log.Fatal(http.ListenAndServe(":5000", server))
}
