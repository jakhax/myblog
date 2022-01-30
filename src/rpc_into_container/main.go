package main

import (
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "server" && os.Args[1] != "client") {
		log.Fatal("Usage: ./main <server | client>")
	}

	if os.Args[1] == "server" {
		RunServer()
	} else {
		RunClient()
	}
}
