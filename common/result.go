package common

import (
	"common/biz"
	"franework/msError"
	"github.com/gin-gonic/gin"
	"net/http"
)

/**
这个文件是写返回相关操作
*/

type Result struct {
	Code int `json:"code"`
	Msg  any `json:"msg"`
}

// 失败的标识
// Fail err 最后自己封装一个
func Fail(ctx *gin.Context, err *msError.Error) {
	ctx.JSON(http.StatusOK, Result{
		Code: err.Code,
		Msg:  err.Err.Error(),
	})
}

// 返回成功
func Sucess(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, Result{
		Code: biz.OK,
		Msg:  data,
	})
}
