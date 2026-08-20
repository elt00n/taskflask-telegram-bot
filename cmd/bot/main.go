// Package main содержит точку входа в приложение.
//
// Исполняемая Go-программа всегда начинается с пакета main и функции main.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/config"
	"github.com/elt00n/taskflask-telegram-bot/internal/database"
	postgresrepository "github.com/elt00n/taskflask-telegram-bot/internal/repository/postgres"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
	telegramtransport "github.com/elt00n/taskflask-telegram-bot/internal/transport/telegram"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	location, err := time.LoadLocation(cfg.DefaultTimezone)
	if err != nil {
		log.Fatalf("load timezone: %v", err)
	}

	databasePool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect PostgreSQL: %v", err)
	}
	defer databasePool.Close()

	if err := database.Migrate(ctx, databasePool); err != nil {
		log.Fatalf("migrate PostgreSQL: %v", err)
	}

	taskRepository := postgresrepository.NewTaskRepository(databasePool)
	memberRepository := postgresrepository.NewChatMemberRepository(databasePool)
	userRepository := postgresrepository.NewUserRepository(databasePool)
	chatRepository := postgresrepository.NewChatRepository(databasePool)
	taskService := service.NewTaskService(
		taskRepository,
		memberRepository,
		nil,
		nil,
	)

	telegramBot, err := telegramtransport.New(
		cfg.TelegramBotToken,
		taskService,
		memberRepository,
		userRepository,
		chatRepository,
		location,
	)
	if err != nil {
		log.Fatal("initialize Telegram: token check or connection failed")
	}

	log.Printf(
		"taskflask bot started in %s environment, timezone %s",
		cfg.Environment,
		cfg.DefaultTimezone,
	)
	telegramBot.Start(ctx)
	log.Print("taskflask bot stopped")
}
