package smssync

import (
	"errors"
	"github.com/clouderhem/misync/consts"
	mlog "github.com/clouderhem/misync/utility/log"
	"net/url"
)

const DirName = "sms"

var (
	jsonFilepath = ""
	xlsxFilepath = ""
)

var log mlog.Log

func init() {
	jsonFilepath = createPath("sms.json")
	xlsxFilepath = createPath("sms.xlsx")

	logger, err := mlog.NewFileLog(DirName, createPath("sms.log"))
	if err != nil {
		panic(errors.Join(errors.New("create sms log file error"), err))
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
