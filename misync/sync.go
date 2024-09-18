package misync

import (
	"github.com/clouderhem/micloud"
	notesync "github.com/clouderhem/misync/misync/pull/note"
	"log"
)

func PullAndSave() {
	err := micloud.Init(micloud.Config{CookieFilepath: "/micloud/.micloud_cookie"})
	if err != nil {
		log.Fatal(err)
	}
	err = notesync.PullAndSave(999)
	if err != nil {
		log.Fatal(err)
	}
}
