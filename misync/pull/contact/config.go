package contactsync

import (
	"errors"
	"github.com/clouderhem/misync/consts"
	mlog "github.com/clouderhem/misync/utility/log"
	"net/url"
)

const DirName = "contact"

var (
	jsonFilepath = ""
	xlsxFilepath = ""
)

var log mlog.Log

func init() {
	jsonFilepath = createPath("contact.json")
	xlsxFilepath = createPath("contact.xlsx")

	logger, err := mlog.NewFileLog(DirName, createPath("contact.log"))
	if err != nil {
		panic(errors.Join(errors.New("create contact log file error"), err))
	}
	log = logger
}

func createPath(base string) string {
	filepath, err := url.JoinPath("/", consts.BaseDirName, DirName, base)
	if err != nil {
		panic(errors.Join(errors.New("cannot build file path"), err))
	}
	return filepath
}
