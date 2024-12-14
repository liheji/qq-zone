package controllers

import (
	"github.com/gin-gonic/gin"
	"path/filepath"
	"qq-zone/models/dao/qq"
)

// GetQrImage 获取二维码图片
func GetQrImage(c *gin.Context) {
	file := filepath.Join(qq.QRCODE_SAVE_PATH)
	c.File(file)
}
