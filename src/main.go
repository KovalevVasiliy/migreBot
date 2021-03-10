package main

import (
    "github.com/go-telegram-bot-api/telegram-bot-api"
    "log"
    "os"
)

func main() {
    pdb := getPostgres()
    pdb.AutoMigrate(&HeadacheEntity{})

    token := os.Getenv("TG_BOT_TOKEN")
    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        panic(err)
    }

    if bot == nil {
        panic("bot is nil")
    }

    log.Print(bot.Self.UserName)
    bot.Debug = true

    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, err := bot.GetUpdatesChan(u)
    if err != nil {
        panic(err)
    }

    for update := range updates {
        if update.Message == nil { // ignore any non-Message Updates
            continue
        }
        switch update.Message.Text {
        case "/start":
            setDialogueStateByUserId(UserId(update.Message.From.ID), start)
        }

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
        state := getDialogueStateByUserId(UserId(update.Message.From.ID))

        switch state {
            case start:
                handleStartState(bot, &msg)
            case getPainLevel:
                handleGetPainLevelState(bot, &msg)
            case getDescription:
                handleGetDescriptionState(bot, &msg)
            case getMedicines:
                handleGetMedicinesState(bot, &msg)
            case getMedicinesEfficacy:
                handleGetMedicinesEfficacyState(bot, &msg)
            case sendHeadachesEmail:
                handleSendHeadachesEmailState(bot, &msg)
            case end:
                handleEndState(bot, &msg)
            }
        }
}
