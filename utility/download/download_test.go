package mdownload

import (
	"github.com/clouderhem/misync/misync/pull/comm"
	"testing"
)

func TestDownload(t *testing.T) {
	err := Download("http://127.0.0.1/file", "/tmp/micloud", "test.mkv")
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
