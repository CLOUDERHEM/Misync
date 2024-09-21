package recordingsync

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	recordingmgr "github.com/clouderhem/micloud/micloud/recording"
	"github.com/clouderhem/micloud/micloud/recording/recording"
	"github.com/clouderhem/micloud/utility/parallel"
	"github.com/clouderhem/misync/consts"
	mdownload "github.com/clouderhem/misync/utility/download"
	"github.com/clouderhem/misync/utility/excel"
	mjson "github.com/clouderhem/misync/utility/json"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func PullAndSave(singleLimit int) error {
	var offset = 0
	var rs []recording.Recording
	for {
		recordings, err := recordingmgr.ListRecordings(offset, singleLimit)
		if err != nil {
			log.LogE("cannot pull recordings info", err)
			return err
		}
		offset += singleLimit
		if len(recordings) < singleLimit {
			break
		}
		rs = append(rs, recordings...)
	}
	if len(rs) == 0 {
		return errors.New("no recordings")
	}
	log.LogI("pulled all recordings info, size: ", len(rs))

	err := saveRecordingsAsXlsx(rs)
	if err != nil {
		log.LogE("cannot save recordings as xlsx", err)
	} else {
		log.LogI("saved recordings as xlsx, size: ", len(rs))
	}

	err = savaRecordingsAsJson(rs)
	if err != nil {
		log.LogE("cannot save recordings as json", err)
	} else {
		log.LogI("saved recordings as jsons, size: ", len(rs))
	}

	downloadRecordingFiles(rs)

	return nil
}

func saveRecordingsAsXlsx(rs []recording.Recording) error {
	xlsx, err := excel.NewSingleSheetExcel(xlsxFilepath)
	if err != nil {
		return err
	}
	for i := range rs {
		bytes, err := json.Marshal(rs[i])
		if err != nil {
			return err
		}
		keys, values := mjson.ListKeysAndValues(bytes)
		xlsx.SetHead(keys)
		xlsx.AddStrsRow(values)
	}
	return xlsx.Save()
}

func savaRecordingsAsJson(rs []recording.Recording) error {
	bytes, err := json.Marshal(rs)
	if err != nil {
		return err
	}
	return os.WriteFile(jsonFilepath, bytes, os.ModePerm)
}

func downloadRecordingFiles(rs []recording.Recording) {
	if len(rs) == 0 {
		log.LogI("no recording files to download")
		return
	}
	log.LogI("starting download recording files")
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	outs, errs := parallel.DoParallel[recording.Recording, any](
		rs,
		func(r recording.Recording) (any, error) {
			time.Sleep(time.Second *
				time.Duration((random.Intn(len(rs)/consts.DefaultReqNumInSec+1))+1))

			fileUrl, err := recordingmgr.GetRecordingFileUrl(r.Id)
			if err != nil {
				log.LogE("cannot get recording file url", err)
				return nil, err
			}
			err = mdownload.Download(fileUrl, filesDirName, r.Name+".mp3")
			if err != nil {
				log.LogE("cannot download recording file", err)
				return nil, err
			}
			return nil, nil
		})
	log.LogI("download and save recording files size: ", len(outs), " err size: ", len(errs))

	err := saveFailedErrs(errs)
	if err != nil {
		log.LogE("cannot save failed rs", err)
	}
	log.LogI("saved failed recording files, size: ", len(errs))

	checkFilesSha1(rs)
}

func saveFailedErrs(errs []parallel.ErrOut[recording.Recording]) error {
	if len(errs) == 0 {
		log.LogI("no errs need to be saved")
		return nil
	}
	bytes, err := json.Marshal(errs)
	if err != nil {
		return err
	}
	return os.WriteFile(recordingFailedFilepath, bytes, os.ModePerm)
}

func savaRecordingWithFailuresAsJson(mp map[string]*recording.Recording) error {
	var rs []recording.Recording
	for _, v := range mp {
		if v != nil {
			rs = append(rs, *v)
		}
	}
	if len(rs) == 0 {
		log.LogI("no recordings of files with SHA-1 check failures")
		return nil
	}
	log.LogI("save recordings with SHA-1 check failures, size: ", len(rs))
	bytes, err := json.Marshal(rs)
	if err != nil {
		return err
	}
	return os.WriteFile(recordingSha1FailedFilepath, bytes, os.ModePerm)
}

func checkFilesSha1(rs []recording.Recording) {
	recordingsMap := make(map[string]*recording.Recording)
	for i := range rs {
		recordingsMap[rs[i].Name+".mp3"] = &rs[i]
	}
	defer func() {
		err := savaRecordingWithFailuresAsJson(recordingsMap)
		if err != nil {
			log.LogE("cannot save failed rs", err)
		}
	}()

	log.LogI("starting checking sha1")
	stat, err := os.Stat(filesDirName)
	if err != nil || !stat.IsDir() {
		log.LogE("cannot stat file or is not dir, stop checking sha1", err)
		return
	}
	dir, err := os.OpenFile(filesDirName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.LogE("cannot open file, stop checking sha1", err)
		return
	}
	files, err := dir.ReadDir(-1)
	if err != nil {
		log.LogE("cannot read dir, stop checking sha1", err)
		return
	}
	for i := range files {
		if files[i].IsDir() {
			continue
		}
		file, err := os.Open(filesDirName + "/" + files[i].Name())
		if err != nil {
			log.LogE("cannot open file, skip checking hash", err)
			continue
		}
		hash := sha1.New()
		_, err = io.Copy(hash, file)
		_ = file.Close()
		if err != nil {
			log.LogE("cannot read file, skip checking hash", err)
			continue
		}
		sha1Hash := hash.Sum(nil)
		r, ok := recordingsMap[filepath.Base(files[i].Name())]
		if !ok || r == nil {
			continue
		}
		if r.Sha1 != fmt.Sprintf("%x", sha1Hash) {
			log.LogI("sha1 not match, file: ", files[i].Name())
		} else {
			recordingsMap[files[i].Name()] = nil
		}
	}
}
