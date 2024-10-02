package mdownload

import (
	"net/http"
	"testing"

	"github.com/clouderhem/misync/misync/pull/comm"
)

func TestRangeDownload(t *testing.T) {
	err := RangeDownload("http://127.0.0.1/file", "/tmp/micloud", "test.mkv")
	if err != nil {
		t.Error(err)
	}
	sha1, err := comm.GetFileSha1("/tmp/micloud/test.mkv")
	if err != nil {
		t.Error(err)
	}
	if sha1 != "a1b5f84c514345fb1b890d9850aa773e4b51ec63" {
		t.Error("not same sha1, download err")
	}
}

func TestRawDownload(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/file", nil)
	err := RawDownload(req, "/tmp/micloud", "test.mkv")
	if err != nil {
		t.Error(err)
	}
	sha1, err := comm.GetFileSha1("/tmp/micloud/test.mkv")
	if err != nil {
		t.Error(err)
	}
	if sha1 != "a1b5f84c514345fb1b890d9850aa773e4b51ec63" {
		t.Error("not same sha1, download err")
	}
}
