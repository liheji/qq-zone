package qq

import (
	"fmt"
	"github.com/tidwall/gjson"
	iurl "net/url"
	"os"
	"path/filepath"
	"qq-zone/models/dao/vo"
	"qq-zone/models/dao/ws"
	"qq-zone/utils/filer"
	"qq-zone/utils/helper"
	"qq-zone/utils/logger"
	ihttp "qq-zone/utils/net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AlbumDown struct {
	ticker     *time.Ticker      // 定时器
	whitelist  map[string]bool   // 要下载的相册
	localFiles map[string]string // 当前本地相册已经存在的文件

	ClientID string
	q        *Qzone
	w        *ws.WebSocketHandler
	Drop     bool
}

var (
	mutex       sync.Mutex         // 互斥锁，下载数累加解决竞态
	chans       chan struct{}      // 缓冲信道控制并行下载的任务数
	waiterIn    sync.WaitGroup     // 等待当前相册下载完才能继续下一个相册
	waiterOut   sync.WaitGroup     // 等待所有相片下载完才能继续往下执行
	total       uint64         = 0 // 总数
	addTotal    uint64         = 0 // 新增数
	succTotal   uint64         = 0 // 成功数
	videoTotal  uint64         = 0 // 视频数
	imageTotal  uint64         = 0 // 相片数
	repeatTotal uint64         = 0 // 重复数
	sequence    uint64         = 0 // 正在下载的相册相片的索引位置
)

func NewAlbumDown(w *ws.WebSocketHandler, q *Qzone) *AlbumDown {
	return &AlbumDown{ClientID: w.ClientID, q: q, w: w}
}

func (a *AlbumDown) initResult() {
	total, addTotal, succTotal, repeatTotal, videoTotal, imageTotal, sequence = 0, 0, 0, 0, 0, 0, 0
}

func (a *AlbumDown) OnDownloadInfo(message string) {
	a.w.Send("downloadInfo", time.Now().Format("2006/01/02 15:04:05")+" "+message)
}

func (a *AlbumDown) OnDownloadProcess(message any) {
	a.w.Send("downloadProcess", message)
}

func (a *AlbumDown) OnDownloadSuccess(message any) {
	a.w.Send("downloadSuccess", message)
}

// SpiderAlbumSelf 爬取相册
func (a *AlbumDown) SpiderAlbumSelf() error {
	task := 10      // "请输入1~100之间的下载并行任务数，默认为1："
	exclude := true // "是否开启防重复下载，可选[y/n]，默认是y："
	var albums = strings.Split(a.q.Album, "$$")

	// 指定要下载的相册
	a.whitelist = make(map[string]bool)
	for _, name := range albums {
		a.whitelist[name] = true
	}

	chans = make(chan struct{}, task)

	a.OnDownloadInfo(fmt.Sprintf("登录成功，欢迎%s，%s", a.q.Nickname, "程序即将开始下载"))

	a.initResult() // 初始化结果
	if err := a.readyDownload(a.q.QQ, exclude); err != nil {
		return err
	}

	info := vo.NewSuccessInfo(a.q.QQ, total, succTotal, imageTotal, videoTotal, addTotal, total-succTotal, repeatTotal)
	a.OnDownloadSuccess(info)

	close(chans)
	a.ticker.Stop()
	return nil
}

// SpiderAlbumAll 爬取所有相册
func (a *AlbumDown) SpiderAlbumAll() error {
	task := 16      // "请输入1~100之间的下载并行任务数，默认为1："
	exclude := true // "是否开启防重复下载，可选[y/n]，默认是y："
	var albums = strings.Split(a.q.Album, "$$")

	// 指定要下载的相册
	a.whitelist = make(map[string]bool)
	for _, name := range albums {
		a.whitelist[name] = true
	}

	chans = make(chan struct{}, task)

	a.OnDownloadInfo(fmt.Sprintf("登录成功，欢迎%s，%s", a.q.Nickname, "程序即将开始下载"))

	var successInfoList []vo.SuccessInfo
	var errRes error = nil
	friendQQ := strings.Split(a.q.FriendQQ, "$$")
	for _, fqq := range friendQQ {
		a.initResult() // 初始化结果
		err := a.readyDownload(fqq, exclude)

		// 删除本次访问好友空间痕迹
		_ = a.delVisitRecord(fqq)

		if err != nil {
			errRes = err
		}

		info := vo.NewSuccessInfo(fqq, total, succTotal, imageTotal, videoTotal, addTotal, total-succTotal, repeatTotal)
		successInfoList = append(successInfoList, *info)

		// 睡眠 N 秒再进行下一个账号
		time.Sleep(time.Second * 3)
	}
	a.OnDownloadSuccess(successInfoList)
	close(chans)
	a.ticker.Stop()
	return errRes
}

func (a *AlbumDown) readyDownload(friendQQ string, exclude bool) error {
	var (
		uin     = a.q.QQ
		hostUin = friendQQ
		gtk     = a.q.Gtk
		cookie  = a.q.Cookie
	)

	if hostUin == "" {
		hostUin = uin
	}

	header := make(map[string]string)
	header["cookie"] = cookie
	header["user-agent"] = USER_AGENT

	go a.heartbeat(GetAlbumListUrl(hostUin, uin, gtk), header)

	albums, err := GetAlbumList(hostUin, uin, gtk, cookie)
	if err != nil {
		return err
	}

	if len(albums) < 1 {
		return fmt.Errorf("该账号( %v )没有可访问的相册", hostUin)
	}

	for _, album := range albums {
		if a.Drop {
			return fmt.Errorf("用户取消下载")
		}

		// 跳过不在白名单中的相册
		name := album.Get("name").String()
		if len(a.whitelist) > 0 {
			if _, ok := a.whitelist[name]; !ok {
				continue
			}
			// 报错给用户
			if album.Get("allowAccess").Int() == 0 {
				a.OnDownloadInfo(fmt.Sprintf("无权访问( %v )的相册( %v )，跳过", hostUin, album.Get("name")))
				continue
			}
		}

		// 直接跳过没有权限的
		if album.Get("allowAccess").Int() == 0 {
			continue
		}

		baseDir := fmt.Sprintf("./storage/qzone/%v/album/", hostUin)
		if name[len(name)-1:] == "." {
			name = strings.ReplaceAll(name, ".", "")
		}

		apath := strings.Trim(baseDir+name, " ")

	RetryCreateDir:
		err := os.MkdirAll(apath, os.ModePerm)
		if err != nil {
			apath = fmt.Sprintf("%v%v", baseDir, helper.Md5(name)[8:24])
			goto RetryCreateDir
		}

		photos, err := GetPhotoList(hostUin, uin, &cookie, gtk, album)
		if err != nil {
			return err
		}

		photoTotal := len(photos)
		total += uint64(photoTotal) // 累加相片/视频总数

		if exclude {
			a.localFiles = make(map[string]string, 0)
			files, _ := filer.GetAllFiles(apath)
			for _, path := range files {
				filename := filepath.Base(path)
				filename = filename[:strings.LastIndex(filename, ".")]
				a.localFiles[filename] = path
			}
		} else {
			_ = os.RemoveAll(apath) // 把当前本地相册删掉重新创建空相册然后下载文件，相当于清空目录资源
			_ = os.MkdirAll(apath, os.ModePerm)
		}

		sequence = 0 // 重新初始化为0

		// 正在下载处理
		for key, photo := range photos {
			if a.Drop {
				return fmt.Errorf("用户取消下载")
			}

			waiterIn.Add(1)
			waiterOut.Add(1)
			chans <- struct{}{}
			go a.StartDownload(hostUin, uin, gtk, cookie, key, photo, album, apath, photoTotal, exclude)
		}

		waiterIn.Wait() // 等待当前相册相片下载完之后才能继续下载下一个相册
	}

	waiterOut.Wait()

	return nil
}

func (a *AlbumDown) StartDownload(hostUin, uin, gtk, cookie string, key int, photo, album gjson.Result, apath string, photoTotal int, exclude bool) {
	defer func() {
		<-chans
		waiterIn.Done()
		waiterOut.Done()

		if err := recover(); err != nil {
			// 打印栈信息
			info := vo.NewProcessInfo(
				sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
				photo.Get("name").String(), "", "", string(logger.PanicTrace()),
			)
			a.OnDownloadProcess(info)

			logger.Println(fmt.Sprintf("%v QQ( %v )的相册[%s]第%d个相片/视频下载过程异常，相片/视频名：%v  Panic信息：%v", time.Now().Format("2006/01/02 15:04:05"), hostUin, album.Get("name").String(), (key + 1), photo.Get("name").String(), string(logger.PanicTrace())))
		}
	}()

	header := make(map[string]string)
	header["cookie"] = cookie
	header["user-agent"] = USER_AGENT

	sloc := photo.Get("sloc").String()
	// 获取相片/视频拍摄时间
	rawshti := photo.Get("rawshoottime").Value()
	rawShoottime := ""
	if reflect.TypeOf(rawshti).Kind() == reflect.String && rawshti.(string) != "" {
		rawShoottime = rawshti.(string)
	} else {
		rawShoottime = photo.Get("uploadtime").String()
	}

	loc, _ := time.LoadLocation("Local")                                           // 重要：获取时区
	shoottime, _ := time.ParseInLocation("2006-01-02 15:04:05", rawShoottime, loc) // 使用模板在对应时区转化为time.time类型
	shootdate := time.Unix(shoottime.Unix(), 0).Format("20060102150405")
	source, filename := "", ""
	if photo.Get("is_video").Bool() {
		url := fmt.Sprintf("https://h5.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/cgi_floatview_photo_list_v2?g_tk=%v&callback=viewer_Callback&topicId=%v&picKey=%v&cmtOrder=1&fupdate=1&plat=qzone&source=qzone&cmtNum=0&inCharset=utf-8&outCharset=utf-8&callbackFun=viewer&uin=%v&hostUin=%v&appid=4&isFirst=1", gtk, album.Get("id").String(), sloc, uin, hostUin)
		_, b, err := ihttp.Get(url, header)
		if err != nil {
			info := vo.NewProcessInfo(
				sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
				photo.Get("name").String(), url, "", "获取下载链接出错",
			)
			a.OnDownloadProcess(info)

			logger.Println(fmt.Sprintf("%v QQ( %v )的相册[%s]第%d部视频获取下载链接出错，视频名：%s  视频地址：%s  错误信息：%s", time.Now().Format("2006/01/02 15:04:05"), hostUin, album.Get("name").String(), key+1, photo.Get("name").String(), url, err.Error()))
			return
		}

		str := string(b)
		str = str[16:strings.LastIndex(str, ")")]
		if !gjson.Valid(str) {
			info := vo.NewProcessInfo(
				sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
				photo.Get("name").String(), url, "", "获取下载链接出错",
			)
			a.OnDownloadProcess(info)

			logger.Println(fmt.Sprintf("%v QQ( %v )的相册[%s]第%d部视频获取下载链接出错，视频名：%s  视频地址：%s  错误信息：invalid json", time.Now().Format("2006/01/02 15:04:05"), hostUin, album.Get("name").String(), key+1, photo.Get("name").String(), url))
			return
		}

		data := gjson.Parse(str).Get("data")
		videos := data.Get("photos").Array()
		if len(videos) < 1 {
			info := vo.NewProcessInfo(
				sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
				photo.Get("name").String(), url, "", "视频链接不存在",
			)
			a.OnDownloadProcess(info)

			logger.Println(fmt.Sprintf("%v QQ( %v )的相册[%s]第%d部视频链接未找到，视频名：%s  视频地址：%s", time.Now().Format("2006/01/02 15:04:05"), hostUin, album.Get("name").String(), key+1, photo.Get("name").String(), url))
			return
		}

		picPosInPage := data.Get("picPosInPage").Int()
		video := videos[picPosInPage]
		videoInfo := video.Get("video_info").Map()
		status := videoInfo["status"].Int()
		// 状态为2的表示可以正常播放的视频，也就是已经转换并上传在QQ空间服务器上
		if status != 2 {
			info := vo.NewProcessInfo(
				sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
				photo.Get("name").String(), url, "", "视频文件无效",
			)
			a.OnDownloadProcess(info)

			logger.Println(fmt.Sprintf("%v QQ( %v )的相册[%s]第%d个视频文件无效，相片/视频名：%s  相片/视频地址：%s  相册列表页地址：%s", time.Now().Format("2006/01/02 15:04:05"), hostUin, album.Get("name").String(), key+1, photo.Get("name").String(), url, photo.Get("url").String()))
			return
		}

		source = videoInfo["video_url"].String()
		if videoInfo["download_url"].String() != "" {
			source = videoInfo["download_url"].String()
		}

		header["Accept"] = "*/*"
		header["Accept-Encoding"] = "identity;q=1, *;q=0"
		header["Connection"] = "keep-alive"
		header["Host"] = ""
		u, err := iurl.Parse(source)
		if err == nil {
			header["Host"] = u.Host
		}

		header["Range"] = "bytes=0-"
		header["Referer"] = fmt.Sprintf("https://user.qzone.qq.com/%v/infocenter", hostUin)
		header["Sec-Fetch-Dest"] = "video"
		header["Sec-Fetch-Mode"] = "no-cors"
		header["Sec-Fetch-Site"] = "cross-site"

		// 目前QQ空间所有视频都是MP4格式，所以暂时固定后缀名都是.mp4
		filename = fmt.Sprintf("VID_%s_%s_%s.mp4", shootdate[:8], shootdate[8:], helper.Md5(sloc)[8:24])
	} else {
		if raw := photo.Get("raw").String(); raw != "" {
			source = raw
		} else if originUrl := photo.Get("origin_url").String(); originUrl != "" {
			source = originUrl
		} else {
			source = photo.Get("url").String()
		}

		// QQ空间相片有不同的文件后缀名，那么不传后缀名的文件名下载的时候会自动获取到对应的文件扩展名
		filename = fmt.Sprintf("IMG_%s_%s_%s", shootdate[:8], shootdate[8:], helper.Md5(sloc)[8:24])
	}

	// 检查是否启用了防重复下载开关,如果开启就忽略下载已经存在的
	if exclude && len(a.localFiles) > 0 {
		pos := strings.LastIndex(filename, ".")
		tmpName := filename
		if pos != -1 {
			tmpName = filename[:pos]
		}

		if p, ok := a.localFiles[tmpName]; ok {
			// 假如本地已经存在这个文件名，那就匹配文件大小是否一致
			head, err := ihttp.Head(source, header)
			if err != nil {
				// 如果该文件地址失效了那也不要删本地已存在的文件
				return
			} else {
				fs, _ := strconv.ParseInt(head.Get("content-length"), 10, 64)
				fileInfo, _ := os.Stat(p)
				fsize := fileInfo.Size()
				if fs > fsize {
					_ = os.RemoveAll(a.localFiles[tmpName])
				} else {
					mutex.Lock()

					if photo.Get("is_video").Bool() {
						videoTotal++
					} else {
						imageTotal++
					}

					succTotal++
					sequence++
					repeatTotal++

					// 通知跳过
					info := vo.NewProcessInfo(
						sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
						photo.Get("name").String(), source, p,
					)
					a.OnDownloadProcess(info)

					mutex.Unlock()

					return
				}
			}
		}
	}

	target := fmt.Sprintf("%s/%s", apath, filename)
	resp, err := ihttp.Download(source, target, header, 5, 600, false)
	if err != nil {
		// 记录 某个相册 下载失败的相片
		info := vo.NewProcessInfo(
			sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
			photo.Get("name").String(), source, target, err.Error(),
		)
		a.OnDownloadProcess(info)
		return
	} else {
		mutex.Lock()

		succTotal++
		sequence++
		addTotal++
		if photo.Get("is_video").Bool() {
			videoTotal++
		} else {
			imageTotal++
		}

		// 通知跳过
		info := vo.NewProcessInfo(
			sequence, uint64(photoTotal), hostUin, album.Get("name").String(),
			photo.Get("name").String(), source, resp["path"].(string),
		)
		a.OnDownloadProcess(info)

		mutex.Unlock()
	}
}

// GetFriends 获取用户所有的好友
func (a *AlbumDown) GetFriends() ([]gjson.Result, error) {
	qq := a.q.QQ
	gtk := a.q.Gtk
	cookie := a.q.Cookie
	url := fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/r.qzone.qq.com/cgi-bin/tfriend/friend_ship_manager.cgi?uin=%v&do=1&fupdate=1&clean=1&g_tk=%v", qq, gtk)
	header := make(map[string]string)
	header["cookie"] = cookie
	header["user-agent"] = USER_AGENT
	str, err := GetMyFriends(url, header)
	if err != nil {
		return nil, err
	}

	friends := gjson.Parse(str).Array()

	return friends, nil
}

// GetAlbumList 获取对我开放空间权限的相册
func (a *AlbumDown) GetAlbumList() (map[string][]*vo.Album, error) {
	qq := a.q.QQ
	gtk := a.q.Gtk
	cookie := a.q.Cookie
	friendQQ := a.q.FriendQQ
	ch := make(chan int, 10)

	swg := &sync.WaitGroup{}
	mapMutex := sync.Mutex{}

	friends := strings.Split(friendQQ, "$$")
	friendAlbumMap := make(map[string][]*vo.Album)
	for _, val := range friends {
		swg.Add(1)
		ch <- 1
		go func(hostUin string) {
			defer func() {
				<-ch
				swg.Done()

				if err := recover(); err != nil {
					// 打印栈信息
					logger.Println(fmt.Sprintf("%v QQ号：%v  Panic信息：%v", time.Now().Format("2006/01/02 15:04:05"), hostUin, string(logger.PanicTrace())))
				}
			}()

			albums, err := GetAlbumList(hostUin, qq, gtk, cookie)

			// 删除本次访问好友空间痕迹
			_ = a.delVisitRecord(hostUin)

			if err != nil {
				return
			}

			accessAlbumList := make([]gjson.Result, 0)
			for _, album := range albums {
				if album.Get("allowAccess").Int() == 0 {
					continue
				}
				accessAlbumList = append(accessAlbumList, album)
			}

			mapMutex.Lock()
			friendAlbumMap[hostUin] = vo.GjsonToObj[*vo.Album](accessAlbumList)
			mapMutex.Unlock()
		}(val)
	}

	swg.Wait()
	close(ch)

	return friendAlbumMap, nil
}

// GetAccess 获取对我开放空间权限的好友
func (a *AlbumDown) GetAccess() (map[string][]*vo.Album, error) {
	qq := a.q.QQ
	gtk := a.q.Gtk
	cookie := a.q.Cookie
	url := fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/r.qzone.qq.com/cgi-bin/tfriend/friend_ship_manager.cgi?uin=%v&do=1&fupdate=1&clean=1&g_tk=%v", qq, gtk)
	header := make(map[string]string)
	header["cookie"] = cookie
	header["user-agent"] = USER_AGENT
	str, err := GetMyFriends(url, header)
	if err != nil {
		return nil, err
	}

	friends := gjson.Parse(str).Array()
	ch := make(chan int, 10)
	swg := &sync.WaitGroup{}

	mapMutex := sync.Mutex{}
	friendAlbumMap := make(map[string][]*vo.Album)
	for _, val := range friends {
		swg.Add(1)
		ch <- 1
		go func(val gjson.Result) {
			hostUin := val.Get("uin").String()
			nickname := val.Get("name").String()
			defer func() {
				<-ch
				swg.Done()

				if err := recover(); err != nil {
					// 打印栈信息
					logger.Println(fmt.Sprintf("%v QQ号：%v  昵称：%v  Panic信息：%v", time.Now().Format("2006/01/02 15:04:05"), hostUin, nickname, string(logger.PanicTrace())))
				}
			}()

			albums, err := GetAlbumList(hostUin, qq, gtk, cookie)

			// 删除本次访问好友空间痕迹
			_ = a.delVisitRecord(hostUin)

			if err != nil {
				return
			}

			mapMutex.Lock()
			friendAlbumMap[fmt.Sprintf("%v$$%v", hostUin, nickname)] = vo.GjsonToObj[*vo.Album](albums)
			mapMutex.Unlock()
		}(val)
	}

	swg.Wait()
	close(ch)

	return friendAlbumMap, nil
}

// 定时发送心跳，防止cookie过期
func (a *AlbumDown) heartbeat(url string, header map[string]string) {
	a.ticker = time.NewTicker(time.Minute * 10)
	for t := range a.ticker.C {
		t.Format("2006/01/02 15:04:05")
		_, _ = ihttp.Head(url, header)
	}
}

// 删除本次访问好友空间痕迹
func (a *AlbumDown) delVisitRecord(huin string) error {
	gtk := a.q.Gtk
	cookie := a.q.Cookie
	vuin := a.q.QQ

	if vuin == huin {
		return nil
	}

	header := make(map[string]string)
	header["cookie"] = cookie
	header["user-agent"] = USER_AGENT

	params := make(map[string]string)
	params["vuin"] = vuin
	params["huin"] = huin
	params["type"] = "1"
	params["src"] = "0"
	params["entrance"] = "4"
	params["qzreferrer"] = fmt.Sprintf("https://user.qzone.qq.com/%v/infocenter", vuin)
	url := fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/w.qzone.qq.com/cgi-bin/tfriend/friendshow_hide_visitor_onelogin?&g_tk=%v", gtk)
	b, err := ihttp.PostForm(url, params, header)
	if err != nil {
		return err
	}

	str := string(b)
	beginSign := "callback("
	beginSignIndex := strings.LastIndex(str, beginSign)
	if beginSignIndex == -1 {
		return fmt.Errorf("Failed to delete access record")
	}

	endSign := ");"
	endSignIndex := strings.LastIndex(str, endSign)
	json := str[beginSignIndex+len(beginSign) : endSignIndex]
	if !gjson.Valid(json) {
		return fmt.Errorf("invalid json")
	}

	if gjson.Get(json, "ret").Int() != 0 {
		return fmt.Errorf("Failed to delete access record")
	}

	return nil
}
