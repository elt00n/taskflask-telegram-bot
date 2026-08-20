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
	"github.com/elt00n/taskflask-telegram-bot/internal/repository/memory"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
	telegramtransport "github.com/elt00n/taskflask-telegram-bot/internal/transport/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	location, err := time.LoadLocation(cfg.DefaultTimezone)
	if err != nil {
		log.Fatalf("load timezone: %v", err)
	}

	taskRepository := memory.NewTaskRepository()
	memberRepository := memory.NewChatMemberRepository()
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
		location,
	)
	if err != nil {
		log.Fatal("initialize Telegram: token check or connection failed")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Printf(
		"taskflask bot started in %s environment, timezone %s",
		cfg.Environment,
		cfg.DefaultTimezone,
	)
	telegramBot.Start(ctx)
	log.Print("taskflask bot stopped")
}
