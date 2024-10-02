package gallerysync

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	mdownload "github.com/clouderhem/misync/utility/download"

	gallerymgr "github.com/clouderhem/micloud/micloud/gallery"
	"github.com/clouderhem/micloud/micloud/gallery/album"
	"github.com/clouderhem/micloud/micloud/gallery/gallery"
	"github.com/clouderhem/micloud/utility/parallel"
	"github.com/clouderhem/misync/consts"
	"github.com/clouderhem/misync/misync/pull/comm"
	"github.com/clouderhem/misync/utility/excel"
	mjson "github.com/clouderhem/misync/utility/json"
	mlog "github.com/clouderhem/misync/utility/log"
)

// each album dir has own log
var logMap = make(map[string]mlog.Log)

func PullAndSave(albumLimit int, singleLimit int) error {
	albums, err := gallerymgr.ListAlbums(0, albumLimit, false)
	if err != nil {
		log.LogE("cannot get albums, err: ", err)
		return err
	}
	wrappers, errs := ListGalleryWrappers(albums.Albums, singleLimit)
	log.LogI("pulled galleries, albumWrappers len: ", len(wrappers), " errs len: ", len(errs))

	err = savePullGalleriesFailedErrs(errs)
	if err != nil {
		log.LogE("cannot save pull gallery errs", err)
	} else {
		log.LogI("save pull gallery errs, errs len: ", len(errs))
	}

	slices.SortFunc(wrappers, func(a, b AlbumsWrapper) int {
		return len(a.Galleries) - len(b.Galleries)
	})

	var totalGalleriesSize, errGalleriesSize = 0, 0
	for i := range wrappers {
		err := saveGalleriesAsXlsx(getAlbumName(&wrappers[i].Album), wrappers[i].Galleries)
		if err != nil {
			log.LogE("cannot save galleries as xlsx", err)
		} else {
			log.LogI("saved galleries as xlsx, albumId: ", wrappers[i].Album.AlbumId,
				" galleries len: ", len(wrappers[i].Galleries))
		}

		err = saveGalleriesAsJson(getAlbumName(&wrappers[i].Album), wrappers[i].Galleries)
		if err != nil {
			log.LogE("cannot save galleries as json", err)
		} else {
			log.LogI("saved galleries as json, albumId: ", wrappers[i].Album.AlbumId,
				" galleries len: ", len(wrappers[i].Galleries))
		}

		downloadErrs := downloadFilesAndCheck(wrappers[i].Album, wrappers[i].Galleries)
		totalGalleriesSize += len(wrappers[i].Galleries)
		errGalleriesSize += len(downloadErrs)
	}

	log.LogI("download galleries, all galleries size: ", totalGalleriesSize, " err size: ", errGalleriesSize)
	if errGalleriesSize > 0 {
		return errors.New("download was not fully successful")
	}
	return nil
}

func ListGalleryWrappers(albums []album.Album, singleLimit int) ([]AlbumsWrapper,
	[]parallel.ErrOut[album.Album]) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	wrappers, errs := parallel.DoParallel(
		albums,
		func(a album.Album) (AlbumsWrapper, error) {
			time.Sleep(time.Second *
				time.Duration((random.Intn(len(albums)/consts.DefaultReqNumInSec+1))+1))

			timelines, err := getTimeline(a.AlbumId, singleLimit)
			if err != nil {
				log.LogE("cannot get timelines, albumId: "+a.AlbumId, err)
				return AlbumsWrapper{}, err
			}
			var totalGalleries []gallery.Gallery
			for i := range timelines {
				if timelines[i].Count == 0 {
					continue
				}
				galleries, err := gallerymgr.ListGalleries(gallery.GalleriesQuery{
					StartDate: timelines[i].StartDate,
					EndDate:   timelines[i].EndDate,
					PageNum:   0,
					PageSize:  timelines[i].Count,
					AlbumId:   a.AlbumId,
				})
				if err != nil {
					log.LogE(fmt.Sprintf("cannot get galleries, albumId: %v, startDate: %v, endDate: %v, pageSize: %v",
						a.AlbumId, timelines[i].StartDate, timelines[i].EndDate, timelines[i].Count), err)
					return AlbumsWrapper{}, err
				}
				if len(galleries.Galleries) != timelines[i].Count {
					log.LogI("galleries size not match, timelines size: ", timelines[i].Count, "actual result size:", len(galleries.Galleries))
				}
				totalGalleries = append(totalGalleries, galleries.Galleries...)
			}
			return AlbumsWrapper{
				Album:     a,
				Galleries: totalGalleries,
			}, nil
		})
	return wrappers, errs
}

func saveGalleriesAsXlsx(dirName string, rs []gallery.Gallery) error {
	xlsx, err := excel.NewSingleSheetExcel(filepath.Join(galleryDirPath, dirName, xlsxFileName))
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

func saveGalleriesAsJson(dirName string, rs []gallery.Gallery) error {
	bytes, err := json.Marshal(rs)
	if err != nil {
		return err
	}
	path := filepath.Join(galleryDirPath, dirName, jsonFileName)
	createDirIfNeed(path)
	return os.WriteFile(path, bytes, os.ModePerm)
}

func downloadFilesAndCheck(album album.Album, galleries []gallery.Gallery) []gallery.Gallery {
	albumName := getAlbumName(&album)
	log := getLogger(albumName)
	if len(galleries) == 0 {
		log.LogI("no gallery files to download")
		return nil
	}
	filesDirPath := filepath.Join(galleryDirPath, albumName, "files")

	var errs []parallel.ErrOut[gallery.Gallery]
	var successCount int

	log.LogI("starting download gallery files")
	for _, g := range galleries {
		// rate limit
		time.Sleep(time.Millisecond * 400)
		err := downloadGalleryFile(filesDirPath, &g)
		if err != nil {
			errs = append(errs, parallel.ErrOut[gallery.Gallery]{
				In:  g,
				Err: err,
			})
		} else {
			successCount++
		}
	}
	log.LogI("downloaded gallery files, success len: ", successCount, " errs len: ", len(errs))

	err := saveDownloadFailedErrs(albumName, errs)
	if err != nil {
		log.LogE("cannot save download failed galleries", err)
	}
	log.LogI("saved download failed galleries files, size: ", len(errs))

	return checkGalleryFilesSha1(filesDirPath, galleries)
}

func downloadGalleryFile(filesDirPath string, gallery *gallery.Gallery) error {
	if isGalleryFileExist(gallery, filepath.Join(filesDirPath, gallery.FileName)) {
		log.LogI("gallery file already exists, file name: ", gallery.FileName)
		return nil
	}
	storageUrl, err := gallerymgr.GetGalleryStorageUrl(gallery.Id)
	if err != nil {
		log.LogE("cannot get gallery storage url, galleryId: "+gallery.Id, err)
		return err
	}
	file, err := gallerymgr.GetGalleryFile(storageUrl)
	if err != nil {
		log.LogE("cannot get gallery file, galleryId: "+gallery.Id, err)
		return err
	}
	req, err := createDownloadFileReq(file)
	if err != nil {
		log.LogE("cannot create file download request", err)
		return err
	}
	err = mdownload.RawDownload(req, filesDirPath, gallery.FileName)
	if err != nil {
		log.LogE("cannot download gallery file, galleryId: "+gallery.Id, err)
		return err
	}
	return nil
}

func isGalleryFileExist(gallery *gallery.Gallery, targetFilePath string) bool {
	_, err := os.Stat(targetFilePath)
	if err != nil {
		return false
	}
	sha1, err := comm.GetFileSha1(targetFilePath)
	if err != nil {
		return false
	}
	return gallery.Sha1 == sha1
}

func createDownloadFileReq(file gallery.GalleryFile) (*http.Request, error) {
	payload := url.Values{}
	payload.Set("meta", file.Meta)
	req, err := http.NewRequest(http.MethodPost, file.Url, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func saveDownloadFailedErrs(dirName string, errs []parallel.ErrOut[gallery.Gallery]) error {
	return comm.SaveErrOuts(filepath.Join(galleryDirPath, dirName, galleryFailedFileName), errs)
}

func savePullGalleriesFailedErrs(errs []parallel.ErrOut[album.Album]) error {
	return comm.SaveErrOuts(filepath.Join(galleryDirPath, galleryFailedFileName), errs)
}

func savaGalleriesWithSha1FailedAsJson(dirPath string, galleries []gallery.Gallery) error {
	path := filepath.Join(dirPath, gallerySha1FailedFileName)
	createDirIfNeed(path)
	bytes, err := json.Marshal(galleries)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, os.ModePerm)
}

func checkGalleryFilesSha1(filesDirPath string, galleries []gallery.Gallery) (failures []gallery.Gallery) {
	galleriesMap := make(map[string]*gallery.Gallery)
	for i := range galleries {
		galleriesMap[galleries[i].FileName] = &galleries[i]
	}
	group := sync.WaitGroup{}
	group.Add(1)

	go func() {
		defer group.Done()
		log.LogI("starting checking sha1, galleries size: ", len(galleries))
		stat, err := os.Stat(filesDirPath)
		if err != nil || !stat.IsDir() {
			log.LogE("cannot stat path or path is not dir, stop sha1 checks", err)
			return
		}
		dir, err := os.OpenFile(filesDirPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			log.LogE("cannot open dir, stop sha1 checks", err)
			return
		}
		files, err := dir.ReadDir(-1)
		if err != nil {
			log.LogE("cannot read dir, stop sha1 checks", err)
			return
		}

		for i := range files {
			if files[i].IsDir() {
				continue
			}
			fileSha1, err := comm.GetFileSha1(filepath.Join(filesDirPath, files[i].Name()))
			if err != nil {
				log.LogE("cannot get sha1 hash, stop sha1 checks", err)
				continue
			}
			r, ok := galleriesMap[files[i].Name()]
			if !ok || r == nil {
				continue
			}
			if r.Sha1 != fileSha1 {
				log.LogI("sha1 not match, file: ", files[i].Name())
			} else {
				galleriesMap[files[i].Name()] = nil
			}
		}
	}()
	group.Wait()

	for _, v := range galleriesMap {
		if v != nil {
			failures = append(failures, *v)
		}
	}
	if len(failures) == 0 {
		log.LogI("no gallery files with sha1 check failed")
		return nil
	}

	err := savaGalleriesWithSha1FailedAsJson(filepath.Dir(filesDirPath), failures)
	if err != nil {
		log.LogE("cannot save failed galleries", err)
	}
	log.LogI("save galleries with sha1 check falied, size: ", len(galleries))

	return
}

func getAlbumName(album *album.Album) string {
	switch album.AlbumId {
	case "1":
		return "相机"
	case "1000":
		return "私密相册"
	}
	return album.Name
}

func getTimeline(albumId string, singleMaxSize int) ([]Timeline, error) {
	timeline, err := gallerymgr.GetTimeline(albumId)
	if err != nil {
		return nil, err
	}
	var ts []Timeline
	for k, v := range timeline.DayCount {
		ts = append(ts, Timeline{
			StartDate: k,
			EndDate:   k,
			Count:     v,
		})
	}

	if len(ts) == 0 {
		log.LogE("no timeline found, albumId: "+albumId, nil)
		return nil, errors.New("no timeline found")
	}

	slices.SortFunc(ts, func(a, b Timeline) int {
		return a.StartDate - b.StartDate
	})

	// todo not full check
	var sum, lastDate = 0, ts[0].StartDate
	var result []Timeline
	for i := range ts {
		if sum+ts[i].Count > singleMaxSize {
			var t Timeline
			if i == 0 {
				t = Timeline{StartDate: lastDate, EndDate: lastDate, Count: sum}
			} else {
				t = Timeline{StartDate: lastDate, EndDate: ts[i-1].EndDate, Count: sum}
			}
			result = append(result, t)
			sum = 0
			lastDate = ts[i].StartDate
		}
		sum += ts[i].Count
		if i == len(ts)-1 {
			result = append(result, Timeline{StartDate: lastDate, EndDate: ts[i].EndDate, Count: sum})

		}
	}
	return result, nil
}

func initLogIfNeed(dirName string) {
	logDirPath := filepath.Join(galleryDirPath, dirName)
	if _, err := os.Stat(logDirPath); os.IsNotExist(err) {
		err := os.MkdirAll(logDirPath, os.ModePerm)
		if err != nil {
			log.LogE("cannot create log directory", err)
		}
	}
	fileLog, err := mlog.NewFileLog(dirName, filepath.Join(logDirPath, logFileName))
	if err != nil {
		log.LogE("cannot create log file", err)
	}
	logMap[dirName] = fileLog
}

func getLogger(dirName string) mlog.Log {
	initLogIfNeed(dirName)
	if l, ok := logMap[dirName]; ok {
		return l
	}
	return log
}

func createDirIfNeed(path string) {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		log.LogE("cannot create dir, path: "+path, err)
	}
}
