// Package main содержит точку входа в приложение.
//
// Исполняемая Go-программа всегда начинается с пакета main и функции main.
package main

import (
	"log"

	"github.com/elt00n/taskflask-telegram-bot/internal/config"
)

func main() {
	cfg := config.Load()

	log.Printf("taskflask bot started in %s environment", cfg.Environment)
}
