package recordingsync

import (
	"errors"
	"github.com/clouderhem/misync/consts"
	mlog "github.com/clouderhem/misync/utility/log"
	"net/url"
)

const DirName = "recording"

var (
	jsonFilepath = ""
	xlsxFilepath = ""
)

var (
	recordingFailedFilepath     = ""
	recordingSha1FailedFilepath = ""
)

var filesDirName = ""

var log mlog.Log

func init() {
	jsonFilepath = createPath("recording.json")
	xlsxFilepath = createPath("recording.xlsx")

	recordingFailedFilepath = createPath("recording_failed.json")
	recordingSha1FailedFilepath = createPath("recording_failed_sha1.json")

	filesDirName = createPath("recording_files")

	logger, err := mlog.NewFileLog(DirName, createPath("recording.log"))
	if err != nil {
		panic(errors.Join(errors.New("create recording log file error"), err))
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
