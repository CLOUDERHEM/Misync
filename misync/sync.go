package misync

import (
	"github.com/clouderhem/micloud"
	contactsync "github.com/clouderhem/misync/misync/pull/contact"
	notesync "github.com/clouderhem/misync/misync/pull/note"
	smssync "github.com/clouderhem/misync/misync/pull/sms"
	"log"
)

func PullNotesAndSave() {
	err := micloud.Init(micloud.Config{CookieFilepath: "/misync/.micloud_cookie"})
	if err != nil {
		log.Fatal(err)
	}
	err = notesync.PullAndSave(999)
	if err != nil {
		log.Fatal(err)
	}
}

func PullContactsAndSave() {
	err := micloud.Init(micloud.Config{CookieFilepath: "/misync/.micloud_cookie"})
	if err != nil {
		log.Fatal(err)
	}
	err = contactsync.PullAndSave(400)
	if err != nil {
		log.Fatal(err)
	}
}

func PullSmsAndSave() {
	err := micloud.Init(micloud.Config{CookieFilepath: "/misync/.micloud_cookie"})
	if err != nil {
		log.Fatal(err)
	}
	err = smssync.PullAndSave(50)
	if err != nil {
		log.Fatal(err)
	}
}
