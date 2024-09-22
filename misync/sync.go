package misync

import (
	"github.com/clouderhem/micloud"
	contactsync "github.com/clouderhem/misync/misync/pull/contact"
	notesync "github.com/clouderhem/misync/misync/pull/note"
	recordingsync "github.com/clouderhem/misync/misync/pull/recording"

	smssync "github.com/clouderhem/misync/misync/pull/sms"
	"log"
)

func init() {
	err := micloud.Init(micloud.Config{
		MicloudCookieFilePath:   "/misync/.micloud_cookie",
		MiaccountCookieFilePath: "/misync/.miaccount_cookie",
	})
	if err != nil {
		log.Fatal(err)
	}
}

func PullNotesAndSave() {
	err := notesync.PullAndSave(999)
	if err != nil {
		log.Fatal(err)
	}
}

func PullContactsAndSave() {
	err := contactsync.PullAndSave(400)
	if err != nil {
		log.Fatal(err)
	}
}

func PullSmsAndSave() {
	err := smssync.PullAndSave(50)
	if err != nil {
		log.Fatal(err)
	}
}

func PullRecordingsAndSave() {
	err := recordingsync.PullAndSave(20)
	if err != nil {
		log.Fatal(err)
	}
}

func RetryPullRecordingsAndSave() {
	err := recordingsync.RedownloadFailedFiles()
	if err != nil {
		log.Fatal(err)
	}
}
