package smssync

import (
	"encoding/json"
	"errors"
	smsmgr "github.com/clouderhem/micloud/micloud/sms"
	"github.com/clouderhem/micloud/micloud/sms/message"
	"github.com/clouderhem/misync/utility/excel"
	mjson "github.com/clouderhem/misync/utility/json"
	"os"
	"time"
)

func PullAndSave(singleLimit int) error {
	var syncTag = "0"
	var syncThreadTag = "0"
	var messages []message.Messages

	for {
		m, err := smsmgr.ListMessages(syncTag, syncThreadTag, singleLimit)
		if err != nil {
			log.LogE("cannot pull messages", err)
			return err
		}
		log.LogI("single pulled, message size", len(m.Entries))

		messages = append(messages, m)
		if len(m.Entries) < singleLimit {
			break
		}
		syncTag = m.Watermark.SyncTag
		syncThreadTag = m.Watermark.SyncThreadTag

		time.Sleep(500 * time.Millisecond)
	}

	var ms []message.Message
	for i := range messages {
		for j := range messages[i].Entries {
			ms = append(ms, messages[i].Entries[j].Entry)
		}
	}
	if len(ms) == 0 {
		return errors.New("no messages found")
	}
	log.LogI("pulled all messages, size: ", len(ms))

	err := saveMessagesAsXlsx(ms)
	if err != nil {
		log.LogE("cannot save messages as xlsx", err)
	} else {
		log.LogI("saved all messages as xlsx, size: ", len(ms))
	}

	err = savaMessagesAsJson(ms)
	if err != nil {
		log.LogE("cannot save messages as json", err)
	} else {
		log.LogI("saved all messages as json, size: ", len(ms))
	}

	return nil
}

func saveMessagesAsXlsx(ms []message.Message) error {
	xlsx, err := excel.NewSingleSheetExcel(xlsxFilepath)
	if err != nil {
		return err
	}
	for i := range ms {
		bytes, err := json.Marshal(ms[i])
		if err != nil {
			return err
		}
		keys, values := mjson.ListKeysAndValues(bytes)
		xlsx.SetHead(keys)
		xlsx.AddStrsRow(values)
	}
	return xlsx.Save()
}

func savaMessagesAsJson(ms []message.Message) error {
	bytes, err := json.Marshal(ms)
	if err != nil {
		return err
	}
	return os.WriteFile(jsonFilepath, bytes, os.ModePerm)
}
