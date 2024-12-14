package helper

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
func Command(name string, mode int, arg ...string) error {
	cmd := exec.Command(name, arg...)
	// 获取输出对象，可以从该对象中读取输出结果
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err != nil {
		return err
	}
	defer stdout.Close()

	// 运行命令
	if err = cmd.Start(); err != nil {
		return err
	}

	// 从管道中实时获取输出并打印到终端
	for {
		buf := make([]byte, 1024)
		_, err := stdout.Read(buf)
		if mode == 1 {
			fmt.Println(string(buf))
		}
		if err != nil {
			break
		}
	}

	// 等待执行完毕
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// SetExifTime 修改文件拍摄/录制时间
func SetExifTime(filePath string) error {
	dstTime, err := ParseFilename(filepath.Base(filePath))
	if err != nil {
		return err
	}

	// exiftool 可执行文件路径
	exifToolPath := "exiftool"
	if _, err := exec.LookPath(exifToolPath); err != nil {
		return fmt.Errorf("exiftool not found: %v", err)
	}

	// 转化位YYYY:MM:DD HH:MM:SS
	dataStr := dstTime.Format("2006:01:02 15:04:05")

	// 构造 exiftool 命令
	err = Command(exifToolPath, 0, "-DateTimeOriginal="+dataStr, "-overwrite_original", filePath)
	if err != nil {
		return err
	}

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
