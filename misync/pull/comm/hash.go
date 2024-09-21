package comm

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
)

func GetFileSha1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	sha := sha1.New()
	_, err = io.Copy(sha, file)
	_ = file.Close()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha.Sum(nil)), nil
}
