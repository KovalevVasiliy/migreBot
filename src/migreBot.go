package main

import (
    "encoding/json"
    "fmt"
    "github.com/360EntSecGroup-Skylar/excelize"
    "github.com/go-telegram-bot-api/telegram-bot-api"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/postgres"
    "gopkg.in/gomail.v2"
    "log"
    "os"
    "regexp"
    "strconv"
    "unicode"
)

type PainLevelEnum uint8

const (
    veryLow PainLevelEnum = iota
    low
    medium
    high
    urgent
)

var PainLevelMap = map[PainLevelEnum]string {
    veryLow : "несильная боль",
    low : "небольшая боль",
    medium : "средняя боль",
    high : "сильная боль",
    urgent : "невыносимая боль",
}

type HeadacheEntity struct {
    gorm.Model
    PainLevel   PainLevelEnum `json:"painLevel"`
    Description string    `json:"description"`
    Medicines   string    `json:"medicines"`
    MedicinesEfficacy bool `json:"medicinesEfficacy"`
    ClientId UserId `json:"userId"`
}

type DialogueState uint8

const (
    start DialogueState = iota
    getPainLevel
    getDescription
    getMedicines
    getMedicinesEfficacy
    sendHeadachesEmail
    end
)

type UserId int

func getDialogueStateByUserId(userId UserId) DialogueState {
    userIdStr := strconv.Itoa(int(userId))
    rStates, _, ctx := getRedisAndContext()
    stateFromRedis, err := rStates.Get(ctx, userIdStr).Result()

    if err != nil {
        log.Print("Return default. ", err)
        return start
    } else {
        state, err := strconv.Atoi(stateFromRedis)
        if err != nil {
            log.Print("Return default. ", err)
            return start
        }
        return DialogueState(state)
    }
}

func setDialogueStateByUserId(userId UserId, state DialogueState) {
    userIdStr := strconv.Itoa(int(userId))
    stateStr := strconv.Itoa(int(state))
    rStates, _, ctx := getRedisAndContext()

    err := rStates.Set(ctx, userIdStr, stateStr, 0).Err()
    if err != nil {
        panic(err)
    }
}

func handleStartState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    switch message.Text {
    default:
        fallthrough
    case "/start":
        msg := tgbotapi.NewMessage(message.ChatID, "Я мигребот! Вы можете использовать"+
            "меня для сохранения истории ваших головных болей")
        headacheButton := tgbotapi.KeyboardButton{
            Text:            "Хочу записать головную боль",
            RequestContact:  false,
            RequestLocation: false,
        }
        queryButton := tgbotapi.KeyboardButton{
            Text:            "Хочу получить выписку",
            RequestContact:  false,
            RequestLocation: false,
        }

        keyBoard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{headacheButton, queryButton})
        msg.ReplyMarkup = keyBoard
        _, _ = bot.Send(msg) // do nothing in case of error
    case "Хочу получить выписку":
        msg := tgbotapi.NewMessage(message.ChatID, "Введите адрес почты, куда будет отправлен дневник")
        msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
            RemoveKeyboard: true,
            Selective:      false,
        }
        _, err := bot.Send(msg)
        if err == nil {
            setDialogueStateByUserId(UserId(message.ChatID), sendHeadachesEmail)
        }

    case "Хочу записать головную боль":
        msg := tgbotapi.NewMessage(message.ChatID, "Оцените вашу боль от 1 до 5, где 1 - "+
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
        keyBoard := tgbotapi.NewReplyKeyboard(keyboard)
        msg.ReplyMarkup = keyBoard
        _, err := bot.Send(msg)
        if err == nil {
            setDialogueStateByUserId(UserId(message.ChatID), getPainLevel)
        }
    }
}

func handleGetPainLevelState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    rText := []rune(message.Text)
    if len(rText) == 1 && unicode.IsDigit(rText[0]) {
        painLevel, _ := strconv.Atoi(message.Text)
        msg := tgbotapi.NewMessage(message.ChatID, "Вы оценили головную боль на " + message.Text + ". " +
            "Расскажите, как это было?")
        msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
            RemoveKeyboard: true,
            Selective:      false,
        }
        _, err := bot.Send(msg)
        if err != nil {
            return
        }
        _, rHeadAches, ctx := getRedisAndContext()

        headAche := HeadacheEntity{PainLevel: PainLevelEnum(painLevel),
            ClientId:UserId(message.ChatID)}
        headAcheBytes, err := json.Marshal(headAche)
        if err != nil {
            panic(err)
        }
        userIdStr := strconv.Itoa(int(message.ChatID))
        err = rHeadAches.Set(ctx, userIdStr, string(headAcheBytes), 0).Err()
        if err != nil {
            panic(err)
        }
        setDialogueStateByUserId(UserId(message.ChatID), getDescription)
    }
}

func handleGetDescriptionState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    _, rHeadAches, ctx := getRedisAndContext()
    userIdStr := strconv.Itoa(int(message.ChatID))
    headAche, err := rHeadAches.Get(ctx, userIdStr).Result()
    if err != nil {
        panic(err)
    }
    var HeadAcheEntity HeadacheEntity
    err = json.Unmarshal([]byte(headAche), &HeadAcheEntity)
    if err != nil {
        panic(err)
    }
    log.Printf("Read headAche from redis: %+v\n", HeadAcheEntity)
    HeadAcheEntity.Description = message.Text
    log.Printf("Updated headAche to redis: %+v\n", HeadAcheEntity)
    headAcheBytes, err := json.Marshal(HeadAcheEntity)
    if err != nil {
        panic(err)
    }
    err = rHeadAches.Set(ctx, userIdStr, string(headAcheBytes), 0).Err()
    if err != nil {
        panic(err)
    }

    msg := tgbotapi.NewMessage(message.ChatID, "Расскажите, какие лекарства вы принимали?")
    _, err = bot.Send(msg)
    if err == nil {
        setDialogueStateByUserId(UserId(message.ChatID), getMedicines)
    }
}

func handleGetMedicinesState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    _, rHeadAches, ctx := getRedisAndContext()
    userIdStr := strconv.Itoa(int(message.ChatID))
    headAche, err := rHeadAches.Get(ctx, userIdStr).Result()
    if err != nil {
        panic(err)
    }
    var HeadAcheEntity HeadacheEntity
    err = json.Unmarshal([]byte(headAche), &HeadAcheEntity)
    if err != nil {
        panic(err)
    }
    log.Printf("Read headAche from redis: %+v\n", HeadAcheEntity)
    HeadAcheEntity.Medicines = message.Text
    log.Printf("Updated headAche to redis: %+v\n", HeadAcheEntity)
    headAcheBytes, err := json.Marshal(HeadAcheEntity)
    if err != nil {
        panic(err)
    }
    err = rHeadAches.Set(ctx, userIdStr, string(headAcheBytes), 0).Err()
    if err != nil {
        panic(err)
    }

    msg := tgbotapi.NewMessage(message.ChatID, "Помогли ли вам лекарства?")
    keyboard := []tgbotapi.KeyboardButton{
        {
            Text:            "Да",
            RequestContact:  false,
            RequestLocation: false,
        },
        {
            Text:            "Нет",
            RequestContact:  false,
            RequestLocation: false,
        },
    }
    keyBoard := tgbotapi.NewReplyKeyboard(keyboard)
    msg.ReplyMarkup = keyBoard
    _, err = bot.Send(msg)
    if err == nil {
        setDialogueStateByUserId(UserId(message.ChatID), getMedicinesEfficacy)
    }
}

func handleGetMedicinesEfficacyState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    var medicinesHelped bool
    switch message.Text {
    case "Да":
        medicinesHelped = true
    case "Нет":
        medicinesHelped = false
    default:
        msg := tgbotapi.NewMessage(message.ChatID, "Помогли ли вам лекартсва?")
        _, _ = bot.Send(msg)
        return
    }
    _, rHeadAches, ctx := getRedisAndContext()
    userIdStr := strconv.Itoa(int(message.ChatID))
    headAche, err := rHeadAches.Get(ctx, userIdStr).Result()
    if err != nil {
        panic(err)
    }
    var headAcheEntity HeadacheEntity
    err = json.Unmarshal([]byte(headAche), &headAcheEntity)
    if err != nil {
        panic(err)
    }
    log.Printf("Read headAche from redis: %+v\n", headAcheEntity)
    headAcheEntity.MedicinesEfficacy = medicinesHelped
    log.Printf("Updated headAche to redis: %+v\n", headAcheEntity)
    headAcheBytes, err := json.Marshal(headAcheEntity)
    if err != nil {
        panic(err)
    }
    err = rHeadAches.Set(ctx, userIdStr, string(headAcheBytes), 0).Err()
    if err != nil {
        panic(err)
    }

    msg := tgbotapi.NewMessage(message.ChatID, "Запись добавлена в электронный дневник " +
        "головных болей")
    msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
        RemoveKeyboard: true,
        Selective:      false,
    }
    _, err = bot.Send(msg)
    if err == nil {
        setDialogueStateByUserId(UserId(message.ChatID), end)

        pdb := getPostgres()
        pdb.Create(&headAcheEntity)
    }
}


func handleEndState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    setDialogueStateByUserId(UserId(message.ChatID), start)
    handleStartState(bot, message)
}

func handleSendHeadachesEmailState(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
    emailRegexp := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
    isEmailValid := emailRegexp.MatchString(message.Text)

    if !isEmailValid {
        msg := tgbotapi.NewMessage(message.ChatID, "Проверьте адрес на ошибки")
        _, _ = bot.Send(msg)
        return
    }
    userIdStr := strconv.Itoa(int(message.ChatID))

    pdb := getPostgres()
    var headAches []HeadacheEntity
    pdb.Find(&headAches, "client_id = ?", userIdStr)

    fileStream := createReport(headAches)
    result := sendReportToEmail(fileStream, message.Text)
    if !result {
        msg := tgbotapi.NewMessage(message.ChatID, "Произошла ошибка при отправке. Попробуйте снова")
        _, _ = bot.Send(msg)
        return
    }
    msg := tgbotapi.NewMessage(message.ChatID, "Дневник будет отправлен на " + message.Text)
    _, err := bot.Send(msg)
    if err == nil {
        setDialogueStateByUserId(UserId(message.ChatID), end)
    }
}

func sendReportToEmail(fileStream *excelize.File, email string) bool {
    tmpFilename := email + ".xlsx"
    defer os.Remove(tmpFilename)
    if err := fileStream.SaveAs(tmpFilename); err != nil {
        log.Print(err)
        return false
    }
    smtpHost := os.Getenv("SMTP_HOST")
    smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
    smtpUser := os.Getenv("SMTP_USER")
    smtpPassword := os.Getenv("SMTP_PASSWORD")

    m := gomail.NewMessage()
    m.SetHeader("From", smtpUser)
    m.SetHeader("To", email)
    m.SetHeader("Subject", "Дневник головных болей")
    m.Attach(tmpFilename)

    d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

    if err := d.DialAndSend(m); err != nil {
        log.Print(err)
        return false
    }
    return true
}

func createReport(headAches []HeadacheEntity) *excelize.File {
    output := excelize.NewFile()
    sheetName := "Дневник"
    index := output.NewSheet(sheetName)
    output.SetActiveSheet(index)
    headers := []string{
        "ID", "Дата приступа", "Уровень боли",
        "Описание", "Лекарства", "Эффективность лекарств",
    }
    columns := []string {
        "A", "B", "C", "D", "E", "F",
    }
    if len(headers) != len(columns) {
        panic("check createReport")
    }

    for ind, header := range headers {
        cell := columns[ind] + strconv.Itoa(1)
        output.SetCellValue(sheetName, cell, header)
    }

    for ind, headAche := range headAches {
        year, month, day := headAche.CreatedAt.Date()
        dateTime := fmt.Sprintf("%d/%d/%d", day, month, year)
        var medicinesEfficacy string
        if headAche.MedicinesEfficacy {
            medicinesEfficacy = "помогло"
        } else {
            medicinesEfficacy = "не помогло"
        }

        output.SetCellValue(sheetName, "A"+strconv.Itoa(ind + 2), headAche.ID)
        output.SetCellValue(sheetName, "B"+strconv.Itoa(ind + 2), dateTime)
        output.SetCellValue(sheetName, "C"+strconv.Itoa(ind + 2), PainLevelMap[headAche.PainLevel])
        output.SetCellValue(sheetName, "D"+strconv.Itoa(ind + 2), headAche.Description)
        output.SetCellValue(sheetName, "E"+strconv.Itoa(ind + 2), headAche.Medicines)
        output.SetCellValue(sheetName, "F"+strconv.Itoa(ind + 2), medicinesEfficacy)
    }
    return output
}
