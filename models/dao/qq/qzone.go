package qq

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"math/rand"
	"net/http"
	iurl "net/url"
	"os"
	"qq-zone/models/dao/vo"
	"qq-zone/models/dao/ws"
	ihttp "qq-zone/utils/net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Qzone struct {
	ClientID string
	w        *ws.WebSocketHandler
	// 外部数据
	QQ       string
	Album    string
	FriendQQ string
	// 登录数据
	Nickname string
	Gtk      string
	Cookie   string

	// 内部使用
	Redirect    string
	WG          sync.WaitGroup
	Drop        bool
	LoginStatus int
}

const (
	QRCODE_SAVE_PATH = "qrcode.png"
	USER_AGENT       = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36"
)

// NewQzone 登录获取g_tk和cookie参数才能进入相册
func NewQzone(clientID string, w *ws.WebSocketHandler) *Qzone {
	return &Qzone{ClientID: clientID, w: w}
}

func (q *Qzone) Login() error {
	q.LoginStatus = 1
	if err := q.loopIfLogin(); err != nil {
		return err
	}
	if err := q.credential(); err != nil {
		return err
	}
	q.LoginStatus = 2
	return nil
}

func (q *Qzone) OnLogin(message string) {
	q.w.Send("login", time.Now().Format("2006/01/02 15:04:05")+" "+message)
}

func (q *Qzone) ResponseQrImage() {
	q.WG.Done()
}

// 循环检查用户是否扫描成功以及是否登录成功
func (q *Qzone) loopIfLogin() error {
	loginSig, err := q.getLoginSig()
	if err != nil {
		return err
	}

	header, err := q.getQRC()
	if err != nil {
		return err
	}

	qrsig := strings.Replace(strings.Split(header.Get("Set-Cookie"), ";")[0], "qrsig=", "", 1)
	ptqrtoken := q.ptqrtoken(qrsig)

	var isFirstLoop bool

	// 响应请求
	q.ResponseQrImage()
OuterLoop:
	for {
		if q.Drop {
			return errors.New("用户取消登录")
		}
		str, err := q.ifLogin(ptqrtoken, loginSig, qrsig)
		if err != nil {
			return err
		}

		if !strings.Contains(str, "") {
			return errors.New("未知错误001，请刷新重试！")
		}

		// 间隔3秒循环一次
		if isFirstLoop {
			time.Sleep(time.Second * 3)
		}

		s := strings.Split(strings.ReplaceAll(str[strings.Index(str, "(")+1:len(str)-1], "'", ""), ",")
		// 65 二维码已失效 66 二维码未失效 67 已扫描,但还未点击确认 0  已经点击确认,并登录成功
		switch s[0] {
		case "65":
			return errors.New("二维码已失效，请重试")

		case "66":
			q.OnLogin("二维码已生成，请使用手机QQ扫码登录")
			isFirstLoop = true
			continue OuterLoop

		case "67":
			q.OnLogin("已扫描，请点击允许登录")
			isFirstLoop = true
			continue OuterLoop

		case "0":
			// 已经点击确认,并登录成功
			q.Nickname = s[len(s)-2]
			q.Redirect = s[2]
			break OuterLoop

		default:
			return errors.New("未知错误002，请重试")
		}
	}

	return nil
}

// 检查用户是否扫描成功以及是否登录成功
func (q *Qzone) ifLogin(ptqrtoken string, loginSig string, qrsig string) (string, error) {
	header := make(map[string]string)
	header["user-agent"] = USER_AGENT
	header["cookie"] = fmt.Sprintf("qrsig=%s;", qrsig)
	url := fmt.Sprintf("https://ssl.ptlogin2.qq.com/ptqrlogin?u1=%s&ptqrtoken=%v&ptredirect=0&h=1&t=1&g=1&from_ui=1&ptlang=2052&action=%v&js_ver=21010623&js_type=1&login_sig=%v&pt_uistyle=40&aid=549000912&daid=5&has_onekey=1", iurl.QueryEscape("https://qzs.qq.com/qzone/v5/loginsucc.html?para=izone"), ptqrtoken, q.action(), loginSig)

	_, b, err := ihttp.Get(url, header)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return string(b), nil
}

// 随机数
func (q *Qzone) t() string {
	return strconv.FormatFloat(rand.Float64(), 'g', -1, 64)
}

// 获取二维码
func (q *Qzone) getQRC() (http.Header, error) {
	url := "https://ssl.ptlogin2.qq.com/ptqrshow?appid=549000912&e=2&l=M&s=3&d=72&v=4&t=" + q.t() + "&daid=5&pt_3rd_aid=0"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	file, err := os.OpenFile(QRCODE_SAVE_PATH, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, err
	}

	return resp.Header, nil
}

// 获取login_sig参数
func (q *Qzone) getLoginSig() (string, error) {
	url := "https://xui.ptlogin2.qq.com/cgi-bin/xlogin?proxy_url=https://qzs.qq.com/qzone/v6/portal/proxy.html&daid=5&&hide_title_bar=1&low_login=0&qlogin_auto_login=1&no_verifyimg=1&link_target=blank&appid=549000912&style=22&target=self&s_url=https://qzs.qq.com/qzone/v5/loginsucc.html?para=izone&pt_qr_app=手机QQ空间&pt_qr_link=https://z.qzone.com/download.html&self_regurl=https://qzs.qq.com/qzone/v6/reg/index.html&pt_qr_help_link=https://z.qzone.com/download.html&pt_no_auth=0"
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.New(err.Error())
	}
	resp.Body.Close()

	setCookies := resp.Header.Values("Set-Cookie")
	if len(setCookies) < 1 {
		return "", errors.New("获取login_sig参数错误，请稍后重试")
	}

	var loginSig string
	for _, val := range setCookies {
		if strings.Contains(val, "pt_login_sig=") {
			s := strings.Split(val, ";")
			for _, v := range s {
				if strings.Contains(v, "pt_login_sig=") {
					loginSig = strings.Replace(v, "pt_login_sig=", "", 1)
				}
			}
		}
	}

	if loginSig == "" {
		return "", errors.New("获取login_sig参数错误，请稍后重试")
	}

	return loginSig, nil
}

/**
 * 获获取ptqrttoken参数
 * header http.Header 将获取二维码接口的headers传进来
 */
func (q *Qzone) ptqrtoken(qrsig string) string {
	e := 0
	for i := 0; i < len(qrsig); i++ {
		e += (e << 5) + int(qrsig[i])
	}

	return strconv.Itoa(2147483647 & e)
}

// 获取action参数
func (q *Qzone) action() string {
	return fmt.Sprintf("0-0-%d", time.Now().Unix()*1000)
}

// 登录成功，验证进入空间的签名
func (q *Qzone) credential() error {
	req, err := http.NewRequest("GET", q.Redirect, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var (
		p_skey string
		needs  = []string{"uin", "skey", "p_uin", "pt4_token", "p_skey"} // 需要从set-cookie取的参数
		cookie = make([]string, 0)
	)

	setCookies := resp.Header.Values("Set-Cookie")
	for _, val := range setCookies {
		c := strings.Split(strings.Split(val, ";")[0], "=")
		name := c[0]
		value := c[1]
		for _, ckey := range needs {
			if name == ckey && value != "" {
				if ckey == "p_skey" {
					p_skey = value
				}
				cookie = append(cookie, fmt.Sprintf("%s=%s", name, value))
			}
		}
	}

	q.Gtk = q.gtk(p_skey)
	q.Cookie = strings.Join(cookie, "; ")

	return nil
}

// 获取登录成功之后的g_tk参数
func (q *Qzone) gtk(skey string) string {
	h := 5381
	for i := 0; i < len(skey); i++ {
		h += (h << 5) + int(skey[i])
	}

	return strconv.Itoa(h & 2147483647)
}

func (q *Qzone) GetSelf() *vo.Friend {
	value, err := strconv.ParseInt(q.QQ, 10, 64)
	if err != nil {
		value = 0
	}
	res := &vo.Friend{
		Uin:   value,
		Name:  "[自己]" + q.Nickname,
		Index: -1,
		Img:   fmt.Sprintf("https://qlogo4.store.qq.com/qzone/%v/%v/50", q.QQ, q.QQ),
	}
	res.Deal()
	return res
}

// GetAlbumListUrl 获取相册列表地址
func GetAlbumListUrl(hostUin string, uin string, gtk string) string {
	return fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/fcg_list_album_v3?g_tk=%v&callback=shine_Callback&hostUin=%v&uin=%v&appid=4&inCharset=utf-8&outCharset=utf-8&source=qzone&plat=qzone&format=jsonp&notice=0&filter=1&handset=4&pageNumModeSort=40&pageNumModeClass=15&needUserInfo=1&idcNum=4&callbackFun=shine", gtk, hostUin, uin)
}

// GetAlbumList 获取相册列表数据
func GetAlbumList(hostUin, uin, gtk, cookie string) ([]gjson.Result, error) {
	headers := make(map[string]string)
	headers["cookie"] = cookie
	headers["user-agent"] = USER_AGENT

	var (
		pageStart int64 = 0
		pageNum   int64 = 30
	)

	var data []gjson.Result

	for {
		url := fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/fcg_list_album_v3?g_tk=%v&callback=shine_Callback&hostUin=%v&uin=%v&appid=4&inCharset=utf-8&outCharset=utf-8&source=qzone&plat=qzone&format=jsonp&notice=0&filter=1&handset=4&pageNumModeSort=40&pageNumModeClass=15&needUserInfo=1&idcNum=4&mode=2&pageStart=%d&pageNum=%d&callbackFun=shine", gtk, hostUin, uin, pageStart, pageNum)
		_, b, err := ihttp.Get(url, headers)
		if err != nil {
			return nil, fmt.Errorf("获取相册列表第%d页出错：%s", pageStart/pageNum+1, err.Error())
		}

		u, err := iurl.Parse(url)
		if err != nil {
			return nil, err
		}

		callbackFunName := u.Query().Get("callbackFun") + "_Callback"
		str := string(b)
		str = str[len(callbackFunName)+1 : strings.LastIndex(str, ")")]
		if !gjson.Valid(str) {
			return nil, fmt.Errorf("相册列表第%d页返回json无效", pageStart/pageNum+1)
		}

		result := gjson.Parse(str)
		if result.Get("code").Int() != 0 {
			return nil, fmt.Errorf("相册列表第%d页错误: %s", pageStart/pageNum+1, gjson.Get(str, "message").String())
		}

		t := result.Get("data")

		if t.Get("albumList").Exists() {
			albumList := t.Get("albumList").Array()
			for _, album := range albumList {
				data = append(data, album)
			}
		}

		nextPageStart := t.Get("nextPageStart").Int()
		if nextPageStart == t.Get("albumsInUser").Int() {
			break
		}

		pageStart = nextPageStart
	}

	return data, nil
}

// GetPhotoList 获取相片列表数据
func GetPhotoList(hostUin, uin string, cookie *string, gtk string, album gjson.Result) ([]gjson.Result, error) {
	headers := make(map[string]string)
	headers["cookie"] = *cookie
	headers["user-agent"] = USER_AGENT

	var (
		pageNum      int64 = 500
		pageStart    int64 = 0
		photoTotal   int64 = 0
		photoPageNum       = 1
		totalInAlbum       = album.Get("total").Int()
	)

	photos := make([]gjson.Result, 0)
	for {
		url := fmt.Sprintf("https://user.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/cgi_list_photo?g_tk=%v&callback=shine_Callback&mode=0&idcNum=4&hostUin=%v&topicId=%v&noTopic=0&uin=%v&pageStart=%v&pageNum=%v&skipCmtCount=0&singleurl=1&batchId=&notice=0&appid=4&inCharset=utf-8&outCharset=utf-8&source=qzone&plat=qzone&outstyle=json&format=jsonp&json_esc=1&callbackFun=shine", gtk, hostUin, album.Get("id").String(), uin, pageStart, pageNum)
		header, body, err := ihttp.Get(url, headers)
		if err != nil {
			return nil, fmt.Errorf("获取相册图片[%s]第%d页错误:%s", album.Get("name").String(), photoPageNum, err.Error())
		}

		var (
			qqPhotoKey string
			setCookie  = header.Get("set-cookie")
		)

		if strings.Contains(setCookie, "qq_photo_key") {
			qqPhotoKey = setCookie[len("qq_photo_key="):strings.Index(setCookie, ";")]
		}

		// 获取qq_photo_key拼接到cookie
		if qqPhotoKey != "" && !strings.Contains(*cookie, "qq_photo_key") {
			*cookie += fmt.Sprintf("; qq_photo_key=%s", qqPhotoKey)
		}

		u, err := iurl.Parse(url)
		if err != nil {
			return nil, err
		}

		callbackFunName := u.Query().Get("callbackFun") + "_Callback"
		str := string(body)
		str = str[len(callbackFunName)+1 : strings.LastIndex(str, ")")]
		if !gjson.Valid(str) {
			return nil, fmt.Errorf("invalid json")
		}

		res := gjson.Parse(str)
		cade := res.Get("code").Int()
		if cade != 0 {
			return nil, fmt.Errorf(res.Get("message").String())
		}

		data := res.Get("data")
		list := data.Get("photoList").Array()
		photos = append(photos, list...)
		if data.Get("totalInPage").Exists() {
			photoTotal += data.Get("totalInPage").Int()
		} else {
			photoTotal += int64(len(list))
		}

		if totalInAlbum == photoTotal { // 说明这个相册下载完成了
			break
		}

		photoPageNum++
		pageStart += 500
	}

	return photos, nil
}

// GetMyFriends 获取我的QQ好友
func GetMyFriends(url string, header map[string]string) (string, error) {
	_, b, err := ihttp.Get(url, header)
	if err != nil {
		return "", fmt.Errorf("获取好友列表出错：%s", err.Error())
	}

	u, err := iurl.Parse(url)
	if err != nil {
		return "", err
	}

	callbackFunName := u.Query().Get("callbackFun") + "_Callback"
	str := string(b)
	str = str[len(callbackFunName)+1 : strings.LastIndex(str, ")")]
	if !gjson.Valid(str) {
		return "", fmt.Errorf("invalid json")
	}

	cade := gjson.Get(str, "code").Int()
	if cade != 0 {
		return "", fmt.Errorf(gjson.Get(str, "message").String())
	}

	friends := gjson.Get(str, "data.items_list")

	return friends.String(), nil
}

// GetCodeMulDelList 删除相片/视频的 codelist 参数
func GetCodeMulDelList(photo gjson.Result) string {
	lloc := photo.Get("lloc").String()
	picrefer := photo.Get("picrefer").String()
	utstr := ""
	if photo.Get("uploadtime").Exists() {
		utstr = photo.Get("uploadtime").String()
	} else {
		utstr = photo.Get("uploadTime").String()
	}

	ut, _ := time.Parse("2006-01-02 15:04:05", utstr)
	uploadtime := ut.Unix()
	forum := photo.Get("forum").String()
	shorturl := ""
	if photo.Get("shorturl").Exists() {
		shorturl = strings.Replace(photo.Get("shorturl").String(), " ", "", -1)
	}

	sloc := photo.Get("sloc").String()
	phototype := photo.Get("phototype").String()
	origin := 0
	if photo.Get("origin").Exists() {
		origin = 1
	}

	return fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v", lloc, picrefer, uploadtime, forum, shorturl, sloc, phototype, origin)
}
