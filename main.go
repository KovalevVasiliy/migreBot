package main


import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strconv"
	"time"
)

type PainLevel uint8

const (
	veryLow PainLevel = 1
	low
	medium
	high
	urgent
)

type HeadacheMHEntity struct {
	startTime   time.Time
	painLevel   PainLevel
	description string
	medicines   string
}

func main() {
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
	var headAche *HeadacheMHEntity = nil
	getDescr := false

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		switch msg.Text {
		case "/start":
			{
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Я мигребот! Вы можете использовать"+
					"меня для сохранения истории ваших головных болей.")
				headacheButton := tgbotapi.KeyboardButton{
					Text:            "Хочу записать головную боль",
					RequestContact:  false,
					RequestLocation: false,
				}

				keyBoard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{headacheButton})
				msg.ReplyMarkup = keyBoard
				_, _ = bot.Send(msg)
			}
		case "Хочу записать головную боль":
			{
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Оцените вашу боль от 1 до 5, где 1 - "+
					"слабая боль, а 5 - ужасная боль:")
				keyboard := []tgbotapi.KeyboardButton{
					{
						Text:            "1",
						RequestContact:  false,
						RequestLocation: false,
					},
					{
						Text:            "2",
						RequestContact:  false,
						RequestLocation: false,
					},
					{
						Text:            "3",
						RequestContact:  false,
						RequestLocation: false,
					},
					{
						Text:            "4",
						RequestContact:  false,
						RequestLocation: false,
					},
					{
						Text:            "5",
						RequestContact:  false,
						RequestLocation: false,
					},
				}
				headAche = new(HeadacheMHEntity)
				headAche.startTime = time.Now()

				keyBoard := tgbotapi.NewReplyKeyboard(keyboard)
				msg.ReplyMarkup = keyBoard
				_, _ = bot.Send(msg)
			}
		case "1":
		case "2":
		case "3":
		case "4":
		case "5": {
			if headAche == nil {
				continue
			}
			painLevel, _ := strconv.Atoi(msg.Text)
			headAche.painLevel = PainLevel(painLevel)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы оценили головную боль на " + msg.Text + "." +
				"Расскажите как это было")

			_, _ = bot.Send(msg)
			getDescr = true

			}
		default: {
			if headAche == nil {
				continue
			}
			if getDescr == true {
				getDescr = false
			} else {
				continue
			}
			headAche.description = msg.Text
			headAcheStr := fmt.Sprintf("Дата приступа: %d-%02d-%02dT%02d:%02d:%02d-00:00 " +
				"Уровень боли: %d. Описание: %s. Препараты: %s",
				headAche.startTime.Year(), headAche.startTime.Month(), headAche.startTime.Day(),
				headAche.startTime.Hour(), headAche.startTime.Minute(), headAche.startTime.Second(), headAche.painLevel,
				headAche.description, headAche.medicines)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Запись сделана\n" + headAcheStr)

			_, _ = bot.Send(msg)

		}
		}
	}
}
