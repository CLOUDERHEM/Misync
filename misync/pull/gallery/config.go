package gallerysync

import (
	"errors"
	"path/filepath"

	"github.com/clouderhem/misync/consts"
	mlog "github.com/clouderhem/misync/utility/log"
)

const DirName = "gallery"

var galleryDirPath = ""

var (
	galleryFailedFileName     = "gallery_failed.json"
	gallerySha1FailedFileName = "gallery_failed_sha1.json"
)

var (
	jsonFileName = "gallery.json"
	xlsxFileName = "gallery.xlsx"
)

var logFileName = "gallery.log"

var log mlog.Log

func init() {
	galleryDirPath = createPath("")
	logger, err := mlog.NewFileLog(DirName, createPath(logFileName))
	if err != nil {
		panic(errors.Join(errors.New("create gallery log file error"), err))
	}
	log = logger
}

func createPath(base string) string {
	// todo
	return filepath.Join("/", consts.BaseDirName, DirName, base)
}
