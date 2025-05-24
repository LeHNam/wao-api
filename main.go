package main

import (
	"log"

	"github.com/LeHNam/wao-api/services/wire"
)

func main() {
	server, err := wire.InitializeServer()
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	//server.AutoMigrate()
	server.SetupRoutes()
	_ = server.Run()
}
