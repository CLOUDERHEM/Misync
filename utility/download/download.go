package mdownload

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/clouderhem/micloud/utility/request"
)

func RangeDownload(url string, dir, filename string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	var offset int64 = 0
	var limit int64 = 1024 * 1024 * 5
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("range",
			fmt.Sprintf("bytes=%v-%v", offset, offset+limit-1))
		resp, err := request.DoRequestNotReadBody(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusBadRequest ||
			resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusNotFound {
			return errors.New(resp.Status)
		}
		read, err := io.Copy(file, resp.Body)
		if err != nil {
			return err
		}
		offset += read
		if resp.ContentLength < limit {
			break
		}
	}
	return nil
}

func RawDownload(req *http.Request, dir, filename string) error {
	_ = os.MkdirAll(dir, os.ModePerm)
	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := request.DoRequestNotReadBody(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
