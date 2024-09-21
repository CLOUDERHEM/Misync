package misync

import "testing"

func TestPullNotesAndSave(t *testing.T) {
	PullNotesAndSave()
}

func TestPullContactsAndSave(t *testing.T) {
	PullContactsAndSave()
}

func TestPullSmsAndSave(t *testing.T) {
	PullSmsAndSave()
}

func TestPullRecordingsAndSave(t *testing.T) {
	PullRecordingsAndSave()
}

func TestRetryPullRecordingsAndSave(t *testing.T) {
	RetryPullRecordingsAndSave()
}
