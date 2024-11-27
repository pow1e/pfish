package utils

import (
	"errors"
	"strings"
)

func CheckByteLength(original, replacement []byte) ([]byte, error) {
	if len(replacement) < len(original) {
		padding := make([]byte, len(original)-len(replacement))
		replacement = append(replacement, padding...)
	} else if len(replacement) > len(original) {
		return nil, errors.New("替换长度超过32，请尝试修改模板和源码后再次生成！")
	}
	return replacement, nil
}

func GetFileExt(fileName string) string {
	if strings.LastIndex(fileName, ".") == -1 {
		return ""
	}
	return fileName[strings.LastIndex(fileName, "."):]
}
