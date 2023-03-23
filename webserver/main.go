package main

import (
	"log"
	"net/http"

	"github.com/kacperf531/sockchat"
)

func main() {

	store, err := sockchat.NewSockChatStore()

	if err != nil {
		log.Fatalf("problem creating file system player store, %v ", err)
	}

	server := sockchat.NewSockChatServer(store)

	if err != nil {
		log.Fatalf("problem creating server %v", err)
	}

	log.Fatal(http.ListenAndServe(":5000", server))
}
