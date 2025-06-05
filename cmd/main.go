package main

import (
	"context"
	"chatrelay/internal/backend"
	"chatrelay/internal/bot"
	"chatrelay/internal/config"
	"chatrelay/internal/telementry"
	"log"
)

func main() {
	config.LoadEnv()

	tp := telemetry.InitTracer()
	defer tp.Shutdown(context.Background())

	go backend.StartMockServer()

	if err := bot.StartSlackBot(); err != nil {
		log.Fatalf("Error running Slack bot: %v", err)
	}
}
