package main

import (
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"tg-receiver-bot/telegram"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	err := os.MkdirAll("./logs", 0666)
	if err != nil {
		log.Fatal(err.Error())
	}
	os.MkdirAll("./logs", 0666)
	f, err := os.OpenFile("./logs/all.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
}

func main() {
	log.Println("creates")
	bot := telegram.New()
	log.Println("starts")
	bot.Start()
	log.Println("works")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Stops work")
	bot.Stop()

}
