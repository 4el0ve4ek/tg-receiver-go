package main

import (
	"log"

	"github.com/joho/godotenv"
	"tg-receiver-bot/telegram"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	bot := telegram.New()
	bot.Start()
	//receivers.NewVkBot()
}

// DONE: parse vk post for videos, photos and text(audio not available)
// TODO: database for checking to whom i should send
