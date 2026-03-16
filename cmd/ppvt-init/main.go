package main

import (
	"context"
	"flag"
	"log"

	"pass-pivot/internal/config"
	toolinit "pass-pivot/internal/tool/init"
)

func main() {
	force := flag.Bool("force", false, "drop and rebuild the target database even when tables already exist")
	flag.Parse()

	cfg := config.LoadInit()
	if err := toolinit.Run(context.Background(), cfg, toolinit.Options{Force: *force}); err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	log.Println("ppvt db init completed")
}
