package msg

import "encoding/json"

func ToByte(message Message) (marshal []byte, err error) {
	marshal, err = json.Marshal(message)
	return
}

func ToObj(marshal []byte, message Message) (err error) {
	err = json.Unmarshal(marshal, message)
	return
}
