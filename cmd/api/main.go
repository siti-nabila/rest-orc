package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/siti-nabila/rest-orc/internal/app"
	"github.com/siti-nabila/rest-orc/internal/config"
)

const configurationPath = "env.yaml"

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(configurationPath)
	if err != nil {
		return fmt.Errorf("load application configuration: %w", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	listenErr := application.Listen(ctx)
	closeErr := application.Close()
	return errors.Join(listenErr, closeErr)
}
