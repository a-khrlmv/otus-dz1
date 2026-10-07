package main

import (
    "log"
)

func main() {

    cfg := config.Load()

    if cfg == nil {
        log.Fatal("Failed to load config")
    }

    log.Printf("Starting %s server on port %s", cfg.AppName, cfg.Port)
    application := app.New(cfg)

    if err := application.Run(); err != nil {
        log.Fatal(err)
    }
}