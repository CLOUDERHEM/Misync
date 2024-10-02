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

func PullAndSave(singlePullLimit int) error {
	var offset = 0
	var rs []recording.Recording
	for {
		recordings, err := recordingmgr.ListRecordings(offset, singlePullLimit)
		if err != nil {
			log.LogE("cannot pull recordings info, err: ", err)
			return err
		}
		log.LogI("single pulled recordings, recordings len: ", len(recordings))

		offset += singlePullLimit
		if len(recordings) < singlePullLimit {
			break
		}
		rs = append(rs, recordings...)
	}
	if len(rs) == 0 {
		return errors.New("no recordings found, stop downloading")
	}
	log.LogI("pulled all recordings info, recordings len: ", len(rs))

	err := saveRecordingsAsXlsx(rs)
	if err != nil {
		log.LogE("cannot save recordings as xlsx", err)
	} else {
		log.LogI("saved recordings as xlsx, recordings len: ", len(rs))
	}

	err = savaRecordingsAsJson(rs)
	if err != nil {
		log.LogE("cannot save recordings as json", err)
	} else {
		log.LogI("saved recordings as json, recordings len: ", len(rs))
	}

	failures := downloadRecordingFiles(rs)
	if len(failures) > 0 {
		log.LogI("downloading recording files has errs, errs len: ", len(failures))

		log.LogI("starting downloading the failures again ")
		downloadRecordingFiles(failures)
	}

	return nil
}

func RedownloadFailedFiles() error {
	log.LogI("starting downloading the failed recording files again")
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
	log.LogI("find failures from local file, failures len: ", len(failures))

	errs := downloadRecordingFiles(failures)
	if len(failures) > 0 {
		log.LogI("re download was not fully successful, failures len: ", len(failures), " errs len: ", len(errs))
		return errors.New("re download was not fully successful")
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
		log.LogI("no recordings that need to be downloaded")
		return nil
	}
	log.LogI("starting download recording files")
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	successes, errs := parallel.DoParallel[recording.Recording, any](
		rs,
		func(r recording.Recording) (any, error) {
			time.Sleep(time.Second *
				time.Duration((random.Intn(len(rs)/consts.DefaultReqNumInSec+1))+1))

			fileUrl, err := recordingmgr.GetRecordingFileUrl(r.Id)
			if err != nil {
				log.LogE("cannot get recording file url, err: ", err)
				return nil, err
			}
			if isRecordingFileExist(r, filesDirName+"/"+r.Name+".mp3") {
				log.LogI("file already exists, file name: ", r.Name)
				return nil, nil
			}
			err = mdownload.RangeDownload(fileUrl, filesDirName, r.Name+".mp3")
			if err != nil {
				log.LogE("cannot download recording file, err: ", err)
				return nil, err
			}
			return nil, nil
		})
	log.LogI("downloaded and saved recording files len: ", len(successes), " errs len: ", len(errs))

	err := saveDownloadFailedErrs(errs)
	if err != nil {
		log.LogE("cannot save the errs of the failed download", err)
	} else {
		log.LogI("saved the errs of the failed download, errs len: ", len(errs))
	}

	return checkRecordingFilesSha1(rs)
}

func isRecordingFileExist(recording recording.Recording, targetFilePath string) bool {
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

func savaRecordingWithSha1FailedAsJson(rs []recording.Recording) error {
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
		stat, err := os.Stat(filesDirName)
		if err != nil || !stat.IsDir() {
			log.LogE("cannot stat file or is not dir, stop sha1 checks", err)
			return
		}
		dir, err := os.OpenFile(filesDirName, os.O_RDONLY, os.ModePerm)
		if err != nil {
			log.LogE("cannot open file, stop sha1 checks", err)
			return
		}
		files, err := dir.ReadDir(-1)
		if err != nil {
			log.LogE("cannot read dir, stop sha1 checks", err)
			return
		}
		log.LogI("starting checking files sha1, recordings len: ", len(rs),
			" local files len: ", len(files))
		for i := range files {
			if files[i].IsDir() {
				continue
			}
			fileSha1, err := comm.GetFileSha1(filesDirName + "/" + files[i].Name())
			if err != nil {
				log.LogE("cannot get sha1 hash, stop sha1 checks", err)
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

	err := savaRecordingWithSha1FailedAsJson(failures)
	if err != nil {
		log.LogE("cannot save recordings with failed sha1 checks", err)
	}
	log.LogI("save recordings with failed sha1 checks, failures len: ", len(failures))

	return failures
}
