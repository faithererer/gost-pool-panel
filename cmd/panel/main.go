package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"gost-pool-panel/internal/buildinfo"
	"gost-pool-panel/internal/panel"
	"gost-pool-panel/internal/store"
)

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.PanelVersion)
		return
	}

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
