package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

type cred struct {
	Username string
	Password string
}

var creds = map[int64]cred{}

type utmail struct {
	Subject string
	Body    string
}

var bot *tgbotapi.BotAPI

type utmails []utmail

var startTime time.Time

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

func main() {
	startTime = time.Now().Local().Add(-1 * time.Hour)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
		fmt.Println("Closing App")
	}()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	timer()
	setupBot()
}

func setupBot() {
	fmt.Println("Setting up the bot...")
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Println("Error registring bot")
		return
	}
	fmt.Println("Connected successfully")

	bot.SetWebhook(tgbotapi.NewWebhook("https://mailut.liara.run/" + bot.Token))

	go http.ListenAndServe(":3000", nil)
	updates := bot.ListenForWebhook("/" + bot.Token)
	for update := range updates {
		handleUpdate(update)
	}
}

// Handler for bot received messages (in telegram)
func handleUpdate(update tgbotapi.Update) {
	text := update.Message.Text
	chatID := update.Message.Chat.ID
	msgID := update.Message.MessageID
	sampleMsg := tgbotapi.NewMessage(chatID,
		`s.aliemami
12345678`)
	if text == "/start" {
		startMsg := tgbotapi.NewMessage(chatID, `
			سلام
همونطور که میدونی برای اینکه بتونم ایمیل هات رو بفرستم اینجا باید یوزرنیم و پسوردت رو بدم به سامانه
برای ادامه کار لطفا یوزرنیم(بدون @) و پسوردت رو توی یه پیام و دو تا خط جدا بهم بده، اینجوری:
		`)
		startMsg.ReplyToMessageID = msgID
		bot.Send(startMsg)
		bot.Send(sampleMsg)

	} else {
		userPass := strings.Split(text, "\r\n")
		if len(userPass) == 1 {
			userPass = strings.Split(text, "\n")
		}
		if len(userPass) != 2 {
			msg := tgbotapi.NewMessage(chatID, `لطفا فرمت زیر رو رعایت کن`)
			msg.ReplyToMessageID = msgID
			bot.Send(msg)
			bot.Send(sampleMsg)
		} else {
			creds[chatID] = cred{Username: userPass[0], Password: userPass[1]}
			log.Println(creds)
			okMsg := tgbotapi.NewMessage(chatID, `حله! ایمیل اومد خبر میدم`)
			okMsg.ReplyToMessageID = msgID
			bot.Send(okMsg)
		}
	}

}

// sends mails to telegram
func sendMailsMessage(chatID int64, mails utmails) {
	for _, mail := range mails {
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
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		bot.Send(msg)
	}
}

// The timer that checks for new email every 5 minutes
func timer() {
	ticker := time.NewTicker(5 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for chatID, cred := range creds {
					mails := getMails(cred.Username, cred.Password)
					go sendMailsMessage(chatID, mails)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func getMails(username string, password string) utmails {

	mails := utmails{}

	// Connect to mail server
	c, err := client.DialTLS("mail.ut.ac.ir:993", nil)
	if err != nil {
		log.Println("Error:", err)
		return mails
	}

	// logout
	defer c.Logout()

	// Login
	if err := c.Login(username, password); err != nil {
		log.Println("Error:", err)
		return mails
	}

	// Select INBOX mailbox
	_, err = c.Select("INBOX", false)
	if err != nil {
		log.Println("Error:", err)
		return mails

	}

	// Search for unseen messages
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	criteria.SentSince = startTime
	uids, err := c.Search(criteria)
	if len(uids) == 0 {
		return utmails{}
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	var section imap.BodySectionName
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(&section)
		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Println("Error #1")
			continue
		}
		header := mr.Header

		// email subject
		var subject string
		subject, err = header.Subject()
		if err != nil {
			log.Println("Error getting subject")
		}

		// Read the mail body
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error:", err)
			return mails

		}
		// email body
		var body string
		switch p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := ioutil.ReadAll(p.Body)
			body = string(b)
		}
		mails = append(mails, utmail{Subject: subject, Body: body})
	}

	if err := <-done; err != nil {
		log.Println("Error:", err)
	}
	return mails
}
