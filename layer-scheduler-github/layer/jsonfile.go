package layer

import (
	"encoding/json"
	"fmt"
	"os"
)

type JsonFile struct {
	filePath string
}

func NewJsonFile(fp string) (*JsonFile, error) {
	return &JsonFile{
		filePath: fp,
	}, nil
}

func (j *JsonFile) Load(src any) (any, error) {
	if !Exists(j.filePath) {
		return nil, fmt.Errorf("文件%s不存在", j.filePath)
	}
	data, err := os.ReadFile(j.filePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, src)
	if err != nil {
		return nil, fmt.Errorf("json 解析失败")
	}
	return src, nil
}

func (j *JsonFile) Dump(src any) error {
	if !Exists(j.filePath) {
		_, err := os.Create(j.filePath)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(src, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(j.filePath, data, 0755)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
