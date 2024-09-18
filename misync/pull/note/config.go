package notesync

import (
	"errors"
	"github.com/clouderhem/misync/consts"
	mlog "github.com/clouderhem/misync/utility/log"
	"net/url"
)

const DirName = "note"

var (
	jsonFilepath = ""
	xlsxFilepath = ""
)

var (
	noteFailedFilepath = ""
	fileFailedFilepath = ""
)

var filesDirName = ""

var log mlog.Log

func init() {
	jsonFilepath = createPath("note.json")
	xlsxFilepath = createPath("note.xlsx")

	noteFailedFilepath = createPath("note_failed.json")
	fileFailedFilepath = createPath("file_failed.json")

	filesDirName = createPath("note_files")

	logger, err := mlog.NewFileLog(DirName, createPath("note.log"))
	if err != nil {
		panic(errors.Join(errors.New("create note log file error"), err))
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
