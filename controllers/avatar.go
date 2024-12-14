package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	service2 "qq-zone/models/service"
	"qq-zone/utils"
)

// GetAvatar 获取二维码图片
func GetAvatar(c *gin.Context) {
	url := c.Param("burl")
	if url == "" {
		c.Data(http.StatusBadRequest, "text/plain", []byte{})
		return
	}

	cache := service2.SingleCache()
	if data, ok := cache.Get(url); ok {
		c.Data(http.StatusOK, "image/png", data.([]byte))
		return
	}

	sourceUrl, err := utils.DecodeBase64(url)
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/plain", []byte{})
		return
	}
	resp, err := utils.GetHttpFile(string(sourceUrl))
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/plain", []byte{})
		return
	}
	cache.SetDefault(url, resp)
	c.Data(http.StatusOK, "image/png", resp)
}
