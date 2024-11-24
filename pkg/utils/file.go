package utils

import (
	"errors"
)

func CheckByteLength(original, replacement []byte) error {
	if len(replacement) < len(original) {
		padding := make([]byte, len(original)-len(replacement))
		replacement = append(replacement, padding...)
	} else if len(replacement) > len(original) {
		return errors.New("替换长度超过32，请尝试修改模板和源码后再次生成！")
	}
	return nil
}
