package mail

import (
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type email struct {
	Subject string
	Body    string
}
type Emails []email

// Cred is UT mail credentials
type Cred struct {
	Username string
	Password string
}

var since time.Time

func init() {
	since = time.Now().Local().Add(-1 * time.Hour)
}

func GetMails(username string, password string) Emails {

	mails := Emails{}

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

	// Search for unseen messages since
	// one hour before running the bot
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	criteria.SentSince = since
	uids, err := c.Search(criteria)
	if len(uids) == 0 {
		return Emails{}
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
		mails = append(mails, email{Subject: subject, Body: body})
	}

	if err := <-done; err != nil {
		log.Println("Error:", err)
	}
	return mails
}
