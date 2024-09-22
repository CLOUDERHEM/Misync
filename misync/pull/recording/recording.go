package recordingsync

import (
	"encoding/json"
	"errors"
	recordingmgr "github.com/clouderhem/micloud/micloud/recording"
	"github.com/clouderhem/micloud/micloud/recording/recording"
	"github.com/clouderhem/micloud/utility/parallel"
	"github.com/clouderhem/misync/consts"
	"github.com/clouderhem/misync/misync/pull/comm"
	mdownload "github.com/clouderhem/misync/utility/download"
	"github.com/clouderhem/misync/utility/excel"
	mjson "github.com/clouderhem/misync/utility/json"
	"io"
	"math/rand"
	"os"
	"sync"
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

	failures := downloadRecordingFiles(rs)
	if len(failures) > 0 {
		// try download again
		downloadRecordingFiles(failures)
	}

	return nil
}

func ReDownloadFromLocalFailFile() error {
	file, err := os.OpenFile(recordingSha1FailedFilepath, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	all, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	var failures []recording.Recording
	err = json.Unmarshal(all, &failures)
	if err != nil {
		return err
	}
	log.LogI("find failures from local file, size: ", len(failures))

	errs := downloadRecordingFiles(failures)
	if len(failures) > 0 {
		log.LogI("retry download, all failure size: ", len(failures), " err size: ", len(errs))
		return errors.New("retry download not all succeed")
	}
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

func downloadRecordingFiles(rs []recording.Recording) []recording.Recording {
	if len(rs) == 0 {
		log.LogI("no recording files to download")
		return nil
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
			if recordingFileExists(r, filesDirName+"/"+r.Name+".mp3") {
				log.LogI("file already exists, file name: ", r.Name)
				return nil, nil
			}
			err = mdownload.Download(fileUrl, filesDirName, r.Name+".mp3")
			if err != nil {
				log.LogE("cannot download recording file", err)
				return nil, err
			}
			return nil, nil
		})
	log.LogI("download and save recording files size: ", len(outs), " err size: ", len(errs))

	err := saveDownloadFailedErrs(errs)
	if err != nil {
		log.LogE("cannot save failed rs", err)
	}
	log.LogI("saved failed recording files, size: ", len(errs))

	return checkRecordingFilesSha1(rs)
}

func recordingFileExists(recording recording.Recording, targetFilePath string) bool {
	_, err := os.Stat(targetFilePath)
	if err != nil {
		return false
	}
	sha1, err := comm.GetFileSha1(targetFilePath)
	if err != nil {
		return false
	}
	return recording.Sha1 == sha1
}

func saveDownloadFailedErrs(errs []parallel.ErrOut[recording.Recording]) error {
	return comm.SaveErrOuts[recording.Recording](recordingFailedFilepath, errs)
}

func savaRecordingWithFailuresAsJson(rs []recording.Recording) error {
	bytes, err := json.Marshal(rs)
	if err != nil {
		return err
	}
	return os.WriteFile(recordingSha1FailedFilepath, bytes, os.ModePerm)
}

func checkRecordingFilesSha1(rs []recording.Recording) []recording.Recording {
	recordingsMap := make(map[string]*recording.Recording)
	for i := range rs {
		recordingsMap[rs[i].Name+".mp3"] = &rs[i]
	}
	group := sync.WaitGroup{}
	group.Add(1)

	go func() {
		defer group.Done()
		log.LogI("starting checking sha1, size: ", len(rs))
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
			fileSha1, err := comm.GetFileSha1(filesDirName + "/" + files[i].Name())
			if err != nil {
				log.LogE("cannot get sha1 hash, stop checking sha1", err)
				continue
			}
			r, ok := recordingsMap[files[i].Name()]
			if !ok || r == nil {
				continue
			}
			if r.Sha1 != fileSha1 {
				log.LogI("sha1 not match, file: ", files[i].Name())
			} else {
				recordingsMap[files[i].Name()] = nil
			}
		}
	}()
	group.Wait()

	var failures []recording.Recording
	for _, v := range recordingsMap {
		if v != nil {
			failures = append(failures, *v)
		}
	}

	err := savaRecordingWithFailuresAsJson(failures)
	if err != nil {
		log.LogE("cannot save failed rs", err)
	}
	log.LogI("save recordings with SHA-1 check failures, size: ", len(rs))

	return failures
}
