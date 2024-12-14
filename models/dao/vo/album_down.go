package vo

import (
	"encoding/base64"
	"encoding/json"
	"github.com/tidwall/gjson"
	"os"
	"path/filepath"
	"qq-zone/utils/filer"
	"qq-zone/utils/helper"
	"sort"
	"time"
)

type ProcessInfo struct {
	Sequence   uint64 `json:"sequence"`
	PhotoTotal uint64 `json:"photoTotal"`

	HostUin   string `json:"hostUin"`
	AlbumName string `json:"albumName"`
	PhotoName string `json:"photoName"`
	Url       string `json:"url"`
	Path      string `json:"path"`

	Size         string `json:"size"`
	SourceDate   string `json:"sourceDate"`
	DownloadDate string `json:"downloadDate"`
	ErrorMsg     string `json:"errorMsg"`
}

func NewProcessInfo(sequence, photoTotal uint64, hostUin, albumName, photoName string, optionAttr ...string) *ProcessInfo {
	d := &ProcessInfo{
		Sequence:     sequence,
		PhotoTotal:   photoTotal,
		HostUin:      hostUin,
		AlbumName:    albumName,
		PhotoName:    photoName,
		DownloadDate: time.Now().Format("2006/01/02 15:04:05"),
	}

	if len(optionAttr) > 0 && optionAttr[0] != "" {
		d.Url = optionAttr[0]
	}
	if len(optionAttr) > 1 && optionAttr[1] != "" {
		path := optionAttr[1]
		filename := filepath.Base(path)
		if sourceTime, err := helper.ParseFilename(filename); err == nil {
			d.SourceDate = sourceTime.Format("2006/01/02")
		}
		if fileInfo, err := os.Stat(path); err == nil {
			d.Size = filer.FormatBytes(fileInfo.Size())
		}
		d.Path = path
	}
	if len(optionAttr) > 2 && optionAttr[2] != "" {
		d.ErrorMsg = optionAttr[2]
	}
	return d
}

type SuccessInfo struct {
	QQ             string `json:"qq"`
	Total          uint64 `json:"total"`
	SuccTotal      uint64 `json:"succTotal"`
	ImageTotal     uint64 `json:"imageTotal"`
	VideoTotal     uint64 `json:"videoTotal"`
	AddTotal       uint64 `json:"addTotal"`
	TotalSuccTotal uint64 `json:"totalSuccTotal"`
	RepeatTotal    uint64 `json:"repeatTotal"`
	DownloadDate   string `json:"downloadDate"`
}

func NewSuccessInfo(qq string, total, succTotal, imageTotal, videoTotal, addTotal, totalSuccTotal, repeatTotal uint64) *SuccessInfo {
	return &SuccessInfo{
		QQ:             qq,
		Total:          total,
		SuccTotal:      succTotal,
		ImageTotal:     imageTotal,
		VideoTotal:     videoTotal,
		AddTotal:       addTotal,
		TotalSuccTotal: totalSuccTotal,
		RepeatTotal:    repeatTotal,
		DownloadDate:   time.Now().Format("2006/01/02 15:04:05"),
	}
}

type SpecialDeal interface {
	Deal()
	Compare(other SpecialDeal) int
}

type Album struct {
	AllowAccess    int    `json:"allowAccess"`
	Anonymity      int    `json:"anonymity"`
	Bitmap         string `json:"bitmap"`
	Classid        int    `json:"classid"`
	Comment        int    `json:"comment"`
	Createtime     int    `json:"createtime"`
	Desc           string `json:"desc"`
	Handset        int    `json:"handset"`
	Id             string `json:"id"`
	Lastuploadtime int    `json:"lastuploadtime"`
	Modifytime     int    `json:"modifytime"`
	Name           string `json:"name"`
	Order          int    `json:"order"`
	Pre            string `json:"pre"`
	Priv           int    `json:"priv"`
	Pypriv         int    `json:"pypriv"`
	Total          int    `json:"total"`
	Viewtype       int    `json:"viewtype"`
}

func (a *Album) Deal() {
	a.Pre = "/avatar/" + base64.StdEncoding.EncodeToString([]byte(a.Pre))
}
func (a *Album) Compare(other SpecialDeal) int {
	otherAlbum, ok := other.(*Album)
	if !ok {
		return 0
	}

	// 比较 AllowAccess，大的在前
	if a.AllowAccess > otherAlbum.AllowAccess {
		return -1
	} else if a.AllowAccess < otherAlbum.AllowAccess {
		return 1
	}

	return 0
}

type Friend struct {
	Uin         int64  `json:"uin"`
	Name        string `json:"name"`
	Index       int    `json:"index"`
	ChangPos    int    `json:"chang_pos"`
	Score       int    `json:"score"`
	SpecialFlag string `json:"special_flag"`
	UncareFlag  string `json:"uncare_flag"`
	Img         string `json:"img"`
}

func (f *Friend) Deal() {
	f.Img = "/avatar/" + base64.StdEncoding.EncodeToString([]byte(f.Img))
}

// 实现 Compare 方法
func (p *Friend) Compare(other SpecialDeal) int {
	// 类型断言
	otherFriend, ok := other.(*Friend)
	if !ok {
		return 0
	}

	// 比较 SpecialFlag，大的在前
	if p.SpecialFlag > otherFriend.SpecialFlag {
		return -1
	} else if p.SpecialFlag < otherFriend.SpecialFlag {
		return 1
	}

	// 比较 UncareFlag，小的在前
	if p.UncareFlag < otherFriend.UncareFlag {
		return -1
	} else if p.UncareFlag > otherFriend.UncareFlag {
		return 1
	}

	// 比较Score，大的在前
	if p.Score > otherFriend.Score {
		return -1
	} else if p.Score < otherFriend.Score {
		return 1
	}

	return 0
}

func GjsonToObj[T SpecialDeal](gList []gjson.Result) []T {
	var objList []T
	for _, val := range gList {
		var obj T
		err := json.Unmarshal([]byte(val.Raw), &obj)
		if err != nil {
			continue
		}
		obj.Deal()
		objList = append(objList, obj)
	}

	// 使用 sort.Slice 进行排序
	sort.SliceStable(objList, func(i, j int) bool {
		return objList[i].Compare(objList[j]) < 0
	})

	return objList
}
