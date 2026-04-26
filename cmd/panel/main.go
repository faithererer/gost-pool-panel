package main

import (
	"log"
	"net/http"

	"gost-pool-panel/internal/panel"
	"gost-pool-panel/internal/store"
)

func main() {
	cfg := panel.LoadConfig()
	st, err := store.New(cfg.DataPath)
	if err != nil {
		log.Fatal(err)
	}
	srv := panel.NewServer(cfg, st)
	srv.StartPoolRuntimes()
	log.Printf("GOST Pool Panel listening on %s", cfg.Listen)
	log.Printf("Open %s", cfg.BaseURL)
	if err := http.ListenAndServe(cfg.Listen, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}
