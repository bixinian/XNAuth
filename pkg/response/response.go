package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type PageData struct {
	List     any   `json:"list"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func JSON(c *gin.Context, httpStatus int, code int, message string, data any) {
	if data == nil {
		data = gin.H{}
	}
	c.JSON(httpStatus, Body{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, CodeOK, "ok", data)
}

func PageOK(c *gin.Context, list any, page int, pageSize int, total int64) {
	OK(c, PageData{
		List:     list,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

func Fail(c *gin.Context, httpStatus int, code int, message string) {
	JSON(c, httpStatus, code, message, nil)
}
