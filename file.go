package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	// 原始 exe 文件路径
	exeFilePath := "E:\\Language\\Go\\Go_code\\src\\pfish\\template\\agent.exe"

	// 读取 exe 文件
	data, err := ioutil.ReadFile(exeFilePath)
	if err != nil {
		log.Fatalf("无法读取文件: %v", err)
	}

	// 替换成你的grpc地址
	original := []byte("exmpaleGrpcServerAddressABCDEFGH")
	replacement := []byte("xxxxxxxxxxxxxxxx") // 替换字符串必须补齐长度
	if len(replacement) < len(original) {
		padding := make([]byte, len(original)-len(replacement))
		replacement = append(replacement, padding...)
	} else if len(replacement) > len(original) {
		log.Fatalf("替换字符串长度过长，无法替换")
	}
	data = bytes.Replace(data, original, replacement, 1)

	// 替换为MD5盐值
	original = []byte("exampleMD51234567891234567891234")
	replacement = []byte("xxxxxxxxxxxxxxx")
	if len(replacement) < len(original) {
		padding := make([]byte, len(original)-len(replacement))
		replacement = append(replacement, padding...)
	} else if len(replacement) > len(original) {
		log.Fatalf("替换字符串长度过长，无法替换")
	}

	modifiedData := bytes.Replace(data, original, replacement, 1)

	// 检查是否进行了替换
	if bytes.Equal(data, modifiedData) {
		log.Println("未找到目标字符串，无需替换")
	} else {
		// 生成新的文件路径
		newExeFilePath := "example.exe"

		// 写入新的 exe 文件
		err = ioutil.WriteFile(newExeFilePath, modifiedData, 0644)
		if err != nil {
			log.Fatalf("无法写入文件: %v", err)
		}

		fmt.Printf("替换成功，生成新的 exe 文件: %s\n", newExeFilePath)
	}
}
