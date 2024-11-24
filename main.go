package main

import (
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/pkg/database"
	"github.com/pow1e/pfish/pkg/log"
	"github.com/pow1e/pfish/server"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	wg sync.WaitGroup
)

func main() {
	// 初始化配置
	var err error
	config.Conf, err = config.Unmarshal("config.yaml")
	if err != nil {
		logrus.Fatal("读取配置文件失败:", err)
	}

	// 初始化log
	log.InitLogger()

	// 初始化db
	database.Init(config.Conf.DataBase)

	// 启动 gRPC 服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.RunGrpcServer()
	}()

	// 启动 Web 服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.WebServer()
	}()

	wg.Wait()
}
