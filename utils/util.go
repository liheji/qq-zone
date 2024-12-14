package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func MIMEs2Ext(mimes []string) string {
	ext := mimes[0]
	for _, mime := range mimes {
		switch mime {
		case ".jpg":
			ext = ".jpg"
		case ".png":
			ext = ".png"
		case ".gif":
			ext = ".gif"
		case ".mp4":
			ext = ".mp4"
		}
	}

	return ext
}

func DecodeBase64(base64Str string) ([]byte, error) {
	// 解码 Base64 字符串
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func StringToObj[T any](payload any) (*T, error) {
	q := new(T)
	switch payload.(type) {
	case map[string]any:
		obj, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(obj, q); err != nil {
			return nil, err
		}
	case []byte:
		if err := json.Unmarshal(payload.([]byte), q); err != nil {
			return nil, err
		}
	case string:
		if err := json.Unmarshal([]byte(payload.(string)), q); err != nil {
			return nil, err
		}
	}
	return q, nil
}

func GetHttpFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	// 检查响应状态码是否为 200
	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("failed to fetch image: status code %d", resp.StatusCode)
	}

	// 读取响应体
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
