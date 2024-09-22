package mdownload

import (
	"errors"
	"fmt"
	"github.com/clouderhem/micloud/utility/request"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func Download(url string, dir, filename string) error {
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
