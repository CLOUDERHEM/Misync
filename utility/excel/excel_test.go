package excel

import "testing"

func TestExcel_AddCellsRow(t *testing.T) {
	excel, err := NewSingleSheetExcel("/tmp/a.xlsx")
	if err != nil {
		t.Error(err)
	}
	excel.SetHead([]string{"sss", "ss", "1sss23"})
	excel.AddStrsRow([]string{"123", "123", "123"})
	err = excel.Save()
	if err != nil {
		t.Error(err)
	}
}
