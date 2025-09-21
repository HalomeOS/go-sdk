package go_sdk

import "testing"

func TestUploadFile(t *testing.T) {
	id, err := UploadFile("D:/test.rar", "1b1427d1c3a88c308a0e0b3d61cf337e", "https://test.wukongyun.fun")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(id)
	}
}
