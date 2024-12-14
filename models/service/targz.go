package service

import (
	"fmt"
	"os"
	"os/exec"
	"qq-zone/utils/helper"
)

// GetQzoneTarGz 打包文件
func GetQzoneTarGz() (string, error) {
	dirPath := "storage/qzone"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %v", err)
	}

	tarFilePath := "storage/qzone.tar.gz"
	cacheStr, err := helper.GetFileDir(dirPath)
	if err != nil {
		return "", err
	}
	cacheKey := fmt.Sprintf("%v,%v", helper.Md5(cacheStr), len(cacheStr))
	if _, ok := SingleCache().Get(cacheKey); ok {
		return tarFilePath, nil
	}

	// 清理之前打包的文件
	if _, err := os.Stat(tarFilePath); err == nil {
		_ = os.RemoveAll(tarFilePath)
	}
	// tar 可执行文件路径
	tarPath := "tar"
	if _, err := exec.LookPath(tarPath); err != nil {
		return "", fmt.Errorf("tar not found: %v", err)
	}

	//  将工作目录切换到 storage
	err = os.Chdir("storage")
	if err != nil {
		return "", fmt.Errorf("failed to change directory: %v", err)
	}
	// 函数结束后切换回原来的目录
	defer os.Chdir("..")

	// 打包 qzone 文件夹的内容
	_, err = helper.Command(tarPath, 0, "-czf", "qzone.tar.gz", "qzone")
	if err != nil {
		return "", err
	}

	SingleCache().SetDefault(cacheKey, "1")

	return tarFilePath, nil
}
