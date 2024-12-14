package helper

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MD5加密
func Md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

/**
 * 生成随机的字符串
 * @param n int 随机字符串长度
 */
func GetRandomString(n int) string {
	s := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for v := range b {
		b[v] = s[rand.Intn(len(s))]
	}
	return string(b)
}

/**
 * gbk编码转utf-8编码
 * @param string s gbk字符串
 */
func GbkToUtf8(s string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewDecoder())
	d, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

/**
 * UTF-8编码转gbk编码
 * @param string s utf-8字符串
 */
func Utf8ToGbk(s string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewEncoder())
	d, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

/**
 * exec 实时获取外部命令的执行输出到终端，参数和系统内置的exec.Command()用法基本一样
 * @param name string 系统内置exec.Command()第一个参数一样
 * @param mode int 运行模式，0：每一条命令执行完毕分别返回一次结果到终端  1：实时获取外部命令的执行输出到终端
 * @param ...string 系统内置exec.Command()第二个参数一样
 */
func Command(name string, mode int, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)

	// 获取输出对象
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout // 将错误重定向到标准输出
	defer stdout.Close()

	// 缓存所有结果
	var outputBuffer bytes.Buffer
	if err = cmd.Start(); err != nil {
		return "", err
	}

	// 根据模式处理输出
	reader := io.TeeReader(stdout, &outputBuffer)
	if mode == 2 {
		// 实时输出到终端
		go func() {
			_, _ = io.Copy(os.Stdout, reader)
		}()
	} else {
		_, _ = io.ReadAll(reader)
	}

	// 等待命令执行完成
	if err = cmd.Wait(); err != nil {
		return "", err
	}

	// 根据模式返回
	if mode == 0 || mode == 1 {
		return outputBuffer.String(), nil
	}
	return "", nil
}

func SetMetadataTime(filePath string) error {
	// 解析文件名中的时间
	dstTime, err := ParseFilename(filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to parse filename time: %v", err)
	}

	// 确认exiftool是否可用
	exifToolPath := "exiftool"
	if _, err := exec.LookPath(exifToolPath); err != nil {
		return fmt.Errorf("exiftool not found: %v", err)
	}

	// 根据文件扩展名选择合适的时间字段
	ext := strings.ToLower(filepath.Ext(filePath))
	var timeFields []string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".tiff", ".webp", ".bmp": // Image
		timeFields = []string{"-DateTimeOriginal", "-CreateDate"}
	case ".mp4", ".mov", ".avi", ".mkv", ".wmv": // Video
		timeFields = []string{"-CreateDate", "-MediaCreateDate", "-MediaModifyDate"}
	case ".mp3", ".wav", ".flac", ".m4a": // Audio
		timeFields = []string{"-CreateDate"}
	case ".gif": // GIF
		timeFields = []string{"-XMP:CreateDate"}
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	// 检查是否有元数据
	output, err := Command(exifToolPath, 1, strings.Join(timeFields, " "), "-s3", filePath)
	if err != nil {
		return fmt.Errorf("failed to read metadata fields: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		return nil
	}

	// 格式化日期为 YYYY:MM:DD HH:MM:SS
	timeStr := dstTime.Format("2006:01:02 15:04:05")

	// 组装命令行参数
	var args []string
	for _, field := range timeFields {
		args = append(args, fmt.Sprintf("%s=%s", field, timeStr))
	}
	args = append(args, "-overwrite_original", filePath)

	_, err = Command(exifToolPath, 0, args...)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %v", err)
	}

	// 更新文件时间戳
	return os.Chtimes(filePath, dstTime, dstTime)
}

// ParseFilename 解析文件名中的日期时间
func ParseFilename(filename string) (time.Time, error) {
	// 定义正则表达式，匹配文件名中的日期时间部分
	re := regexp.MustCompile(`(?:VID|IMG)_(\d{8})_(\d{6})_.*\..+`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return time.Time{}, fmt.Errorf("filename format invalid: %s", filename)
	}

	// 提取日期和时间部分
	datePart := matches[1] // 例如：20190831
	timePart := matches[2] // 例如：211215

	// 将日期和时间部分拼接成完整时间字符串
	dateTimeStr := fmt.Sprintf("%s%s", datePart, timePart)

	// 加载上海时区
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return time.Time{}, err
	}

	// 解析为 time.Time 类型，指定时区
	parsedTime, err := time.ParseInLocation("20060102150405", dateTimeStr, loc)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}

func GetFileDir(root string) (string, error) {
	var result string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %v", path, err)
		}
		// 获取文件或文件夹信息
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %q: %v", path, err)
		}
		// 格式化信息
		if info.IsDir() {
			result += fmt.Sprintf("Directory: %s", path)
		} else {
			result += fmt.Sprintf("File: %s | Size: %d bytes | Modified: %s | Mode: %s",
				path,
				info.Size(),
				info.ModTime().Format(time.RFC3339),
				info.Mode(),
			)
		}
		return nil
	})
	return result, err
}
