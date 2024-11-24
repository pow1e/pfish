package file

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/pkg/model"
	"github.com/pow1e/pfish/pkg/model/response"
	"github.com/xuri/excelize/v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WriteImageFile 根据邮箱写入对应文件 并且返回web访问的邮箱路径
func WriteImageFile(
	conf *config.Server,
	email string,
	clickTime time.Time,
	imgData []byte,
) (string, error) {
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("无法获取当前工作目录: %v", err)
	}
	fmt.Println("当前工作目录:", cwd)

	// 拼接静态目录的绝对路径
	baseDir := filepath.Join(cwd, conf.Static.FilePath)
	fmt.Println("目标静态目录:", baseDir)

	// 检查静态目录是否存在，如果不存在则创建
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		fmt.Println("静态目录不存在，正在创建:", baseDir)
		err = os.MkdirAll(baseDir, 0755) // 创建静态目录
		if err != nil {
			return "", fmt.Errorf("创建静态目录失败 %s: %v", baseDir, err)
		}
	} else {
		fmt.Println("静态目录已存在:", baseDir)
	}

	// 拼接邮箱对应的目录路径
	emailDir := filepath.Join(baseDir, email)
	fmt.Println("目标邮箱目录:", emailDir)

	// 检查邮箱目录是否存在，如果不存在则创建
	if _, err := os.Stat(emailDir); os.IsNotExist(err) {
		fmt.Println("邮箱目录不存在，正在创建:", emailDir)
		err = os.MkdirAll(emailDir, 0755) // 递归创建邮箱目录
		if err != nil {
			return "", fmt.Errorf("创建邮箱目录(%s)失败: %v", emailDir, err)
		}
	} else {
		fmt.Println("邮箱目录已存在:", emailDir)
	}

	// 拼接文件路径 文件名为 邮箱_年-月-日-时-分-秒.png
	filename := fmt.Sprintf("%s_%s.png", email, toBusinessTime(clickTime))
	filePath := filepath.Join(emailDir, filename)
	fmt.Println("文件路径:", filePath)

	// 写入文件
	if err := ioutil.WriteFile(filePath, imgData, 0644); err != nil {
		return "", fmt.Errorf("文件写入失败: %v", err)
	}

	// 生成Web访问路径
	// 确保路径有前缀 /

	webPath := fmt.Sprintf("http://%s:%s/%s/%s/%s?pass={pass}",
		conf.IP,
		conf.Web.Port,
		path.Join(strings.TrimSuffix(conf.Web.Prefix, "/"), strings.TrimPrefix(conf.Static.WebPath, "/")),
		email,
		filename,
	)

	return webPath, nil
}

// toBusinessTime 格式化时间为 年-月-日-时-分-秒
func toBusinessTime(clickTime time.Time) string {
	return clickTime.Format("2006-01-02-15-04-05")
}

func OutputMessagesToExcel(c *gin.Context, messages []*model.Message, email ...string) {
	sheetName := "导出钓鱼结果"
	f := excelize.NewFile()
	sheetIndex, _ := f.NewSheet(sheetName) // 创建新工作表
	f.SetActiveSheet(sheetIndex)           // 设置为活动表
	f.DeleteSheet("Sheet1")                // 删除默认的 Sheet1

	headers := []string{"用户ID", "主机名称", "IP地址", "图片地址", "PID", "进程名", "点击时间"}
	// 设置表头
	for i, header := range headers {
		cell := string('A'+i) + "1"
		f.SetCellValue(sheetName, cell, header)
	}
	// 填充数据
	for rowIndex, message := range messages {
		row := rowIndex + 2
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), message.Uid)
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), message.Computer)
		f.SetCellValue(sheetName, "C"+strconv.Itoa(row), message.Internal)
		f.SetCellValue(sheetName, "D"+strconv.Itoa(row), message.PID)
		f.SetCellValue(sheetName, "E"+strconv.Itoa(row), message.ProcessName)
		f.SetCellValue(sheetName, "F"+strconv.Itoa(row), message.Picture)
		f.SetCellValue(sheetName, "G"+strconv.Itoa(row), message.ClickTime)
	}
	// 保存到内存缓冲区
	buffer := new(bytes.Buffer)
	if err := f.Write(buffer); err != nil {
		c.JSON(http.StatusOK, response.Response{
			Code: 500,
			Msg:  "保存文件失败",
		})
		return
	}

	// 设置 HTTP 响应头并返回文件
	fileName := time.Now().Format("2006年01月02日15时04分05秒") + "-导出结果.xlsx"
	if len(email) > 0 {
		fileName = email[0] + "-" + fileName
	} else {
		fileName = "所有点击事件" + "-" + fileName
	}
	c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", buffer.Bytes())

}
