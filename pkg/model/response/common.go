package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

const (
	SuccessCode = 200
	ErrorCode   = 500
)

func Fail(c *gin.Context) {
	c.JSON(http.StatusOK, &Response{
		Code: ErrorCode,
		Data: "内部错误，请重试",
	})
}

func FailWithMessage(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, &Response{
		Code: ErrorCode,
		Msg:  msg,
	})
}

func OK(c *gin.Context) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
	})
}

func OkWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
		Data: data,
	})
}

func OkWithMessage(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
		Msg:  msg,
	})
}

func OkWithDetail(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
		Msg:  msg,
		Data: data,
	})
}
