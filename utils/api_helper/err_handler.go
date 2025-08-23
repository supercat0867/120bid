package api_helper

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func BadRequestHandler(c *gin.Context, err error) {
	c.JSON(http.StatusOK, Response{Status: 400, Message: err.Error(), ErrCode: 1})
	c.Abort()
	return
}

func InternalServerHandler(c *gin.Context, err error) {
	c.JSON(http.StatusOK, Response{Status: 500, Message: err.Error(), ErrCode: 2})
	c.Abort()
	return
}
