package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"qq-zone/models/service"
	"strings"
	"time"
)

// GetResult 获取结果
func GetResult(c *gin.Context) {
	gzPackage, err := service.GetQzoneTarGz()
	if err != nil {
		c.Data(http.StatusNotFound, "text/plain", []byte{})
		return
	}

	ext := ".tar.gz"
	filename := filepath.Base(gzPackage)
	filePre := strings.ReplaceAll(filename, ext, "")

	fileStat, err := os.Stat(gzPackage)
	if err == nil {
		c.Header("Content-Length", fmt.Sprintf("%d", fileStat.Size()))
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+fmt.Sprintf("%s-%v%s", filePre, time.Now().Format("20060102150406"), ext))

	c.File(gzPackage)
}
