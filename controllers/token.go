package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"qq-zone/models/service"
	"strings"
)

// GetToken 获取认证token
func GetToken(c *gin.Context) {
	genKey := strings.ReplaceAll(uuid.New().String(), "-", "")
	service.SingleCache().SetDefault(genKey, 0)
	c.JSON(200, gin.H{"token": genKey})
}
