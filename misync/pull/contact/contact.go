package contactsync

import (
	"encoding/json"
	"errors"
	contactmgr "github.com/clouderhem/micloud/micloud/contact"
	"github.com/clouderhem/micloud/micloud/contact/contact"
	"github.com/clouderhem/misync/utility/excel"
	mjson "github.com/clouderhem/misync/utility/json"
	"os"
)

func PullAndSave(limit int) error {
	contacts, err := contactmgr.ListContacts(limit)
	if err != nil {
		return err
	}
	var contents []contact.Content
	for _, v := range contacts.Content {
		contents = append(contents, v.Content)
	}
	if len(contents) == 0 {
		return errors.New("no contents found")
	}
	log.LogI("find the contents", len(contents))

	err = saveContentsAsXlsx(contents)
	if err != nil {
		log.LogE("save contact as xlsx failed", err)
	} else {
		log.LogI("save contact as xlsx success, size", len(contents))
	}
	err = savaContentsAsJson(contents)
	if err != nil {
		log.LogE("save contents as json failed", err)
	} else {
		log.LogI("save contents as json success, size", len(contents))
	}

	return nil
}

func saveContentsAsXlsx(contents []contact.Content) error {
	xlsx, err := excel.NewSingleSheetExcel(xlsxFilepath)
	if err != nil {
		return err
	}
	for i := range contents {
		bytes, err := json.Marshal(contents[i])
		if err != nil {
			return err
		}
		keys, values := mjson.ListKeysAndValues(bytes)
		xlsx.SetHead(keys)
		xlsx.AddStrsRow(values)
	}
	return xlsx.Save()
}

func savaContentsAsJson(contents []contact.Content) error {
	bytes, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	return os.WriteFile(jsonFilepath, bytes, os.ModePerm)
}
