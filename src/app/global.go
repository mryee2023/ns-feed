package app

import (
	"encoding/json"
)

var ConfigFilePath string

func ToJson(v interface{}) string {
	b, e := json.Marshal(v)
	if e != nil {
		return ""
	}
	return string(b)
}
