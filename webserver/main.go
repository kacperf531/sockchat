package main

import (
	"log"
	"net/http"

	"github.com/kacperf531/sockchat"
)

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	store, err := sockchat.NewSockChatStore()

	if err != nil {
		log.Fatalf("problem creating file system player store, %v ", err)
	}

	server := sockchat.NewSockChatServer(store)

	log.Fatal(http.ListenAndServe(":5000", server))
}
