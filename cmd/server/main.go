package main

import (
	"context"
	"log"
	"os"

	"github.com/example/hen-quicknotes/internal/app"
)

func main() {
	dataFile := os.Getenv("QUICKNOTES_DATA_FILE")
	server, err := app.NewServer(context.Background(), dataFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HEN-QuickNotes listening on %s with data file %q", server.Addr, dataFile)
	log.Fatal(server.ListenAndServe())
}
