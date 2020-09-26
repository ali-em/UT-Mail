package telegram

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ali-em/UT-Mail/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

var lessons = map[string]string{
	"3991810128301": "سیستم_عامل",
	"3991810139702": "هوش_مصنوعی",
	"3991810120905": "زبان_تخصصی",
	"3991810153803": "سیگنال",
	"3991810153601": "CAD",
	"3991810157401": "کامپایلر",
	"3991810121801": "سیستم_هوشمند",
	"3991810114901": "نرم۱",
}

var bot *tgbotapi.BotAPI

// Maps telegram chatID to email credentials
var users = map[int64]mail.Cred{}

// Setup the bot
func Setup() {
	fmt.Println("Setting up the bot...")
	timer()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	bot, err = tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Println("Error registering bot")
		return
	}
	fmt.Println("Connected successfully")

	bot.SetWebhook(tgbotapi.NewWebhook(os.Getenv("URL") + bot.Token))

	updates := bot.ListenForWebhook("/" + bot.Token)

	go http.ListenAndServe(":3000", nil)

	for update := range updates {
		fmt.Println("Received message")
		handleUpdate(update)
	}
}

const (
	startReply = `
	سلام
همونطور که میدونی برای اینکه بتونم ایمیل هات رو بفرستم اینجا باید یوزرنیم و پسوردت رو بدم به سامانه
برای ادامه کار لطفا یوزرنیم(تا قبل @) و پسوردت رو توی یه پیام و دو تا خط جدا بهم بده، اینجوری:
`
	sampleReply = "s.aliemami\n12345678"
	wrongReply  = "لطفا فرمت زیر رو رعایت کن"
	okReply     = "حله! ایمیل اومد خبر میدم"
)

// Handler for bot received messages (in telegram)
func handleUpdate(update tgbotapi.Update) {
	text := update.Message.Text
	chatID := update.Message.Chat.ID
	msgID := update.Message.MessageID

	sampleMsg := tgbotapi.NewMessage(chatID,
		sampleReply)

	if text == "/start" {
		startMsg := tgbotapi.NewMessage(chatID, startReply)
		startMsg.ReplyToMessageID = msgID
		bot.Send(startMsg)
		bot.Send(sampleMsg)
		return

	}

	// Windows end of line
	userPass := strings.Split(text, "\r\n")
	if len(userPass) == 1 {
		userPass = strings.Split(text, "\n")
	}
	if len(userPass) != 2 {
		wrongMsg := tgbotapi.NewMessage(chatID, wrongReply)
		wrongMsg.ReplyToMessageID = msgID
		bot.Send(wrongMsg)
		bot.Send(sampleMsg)
		return
	}

	users[chatID] = mail.Cred{Username: userPass[0], Password: userPass[1]}
	log.Println(users)
	okMsg := tgbotapi.NewMessage(chatID, okReply)
	okMsg.ReplyToMessageID = msgID
	bot.Send(okMsg)
}

// sends mails to telegram
func sendMailsMessage(chatID int64, maleList mail.Emails) {
	for _, mail := range maleList {
		var topic string
		for c, t := range lessons {
			if strings.Contains(mail.Subject, c) {
				topic = t
				break
			}
		}
		var text string
		if topic != "" {
			text = "#" + topic + "\n\n"
		}
		text += "<b>" + mail.Subject + "</b>\n\n"
		text += mail.Body

		// Telegram has a limit of 4096 bytes for a message
		if len(text) >= 2000 {
			text = text[0:1900] + "\n\n متن طولانی/در ایمیل اصلی بررسی شود"
		}
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		bot.Send(msg)
	}
}

// The timer that checks for new email every 5 minutes
func timer() {
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for chatID, cred := range users {
					go func(id int64, c mail.Cred) {
						mails := mail.GetMails(c.Username, c.Password)
						sendMailsMessage(id, mails)
					}(chatID, cred)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
