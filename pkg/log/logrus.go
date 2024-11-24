package log

import (
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
)

func InitLogger() {
	// 创建日志文件
	logFile, err := os.OpenFile("web.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("[-] 打开文件失败: %v", err)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PadLevelText:    false, // 将确保所有日志级别文本的长度相同。
		FullTimestamp:   true,  // 在日志中显示完整的时间戳
	})
	logrus.SetOutput(io.MultiWriter(os.Stdout, logFile)) // 同时输出到控制台和文件
	logrus.SetLevel(logrus.InfoLevel)                    // 设置日志级别
}
