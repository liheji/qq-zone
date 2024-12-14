package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"qq-zone/models/service"
)

// GetResult 获取结果
func GetResult(c *gin.Context) {
	gzPackage, err := service.GetQzoneTarGz()
	if err != nil {
		c.Data(http.StatusNotFound, "text/plain", []byte{})
		return
	}

	fileStat, err := os.Stat(gzPackage)
	if err == nil {
		c.Header("Content-Length", fmt.Sprintf("%d", fileStat.Size()))
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(gzPackage))

	c.File(gzPackage)
}
