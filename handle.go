package main

import (
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/skip2/go-qrcode"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func GetLocalSessionData() {
	// 获取用户的主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	// 构建 session_data 文件的完整路径
	sessionDataPath := filepath.Join(homeDir, ".bilibili-download", "session_data")
	// 检查文件是否存在
	_, err = os.Stat(sessionDataPath)
	if os.IsNotExist(err) {
		// 文件不存在，返回空字符串
		return
	} else if err != nil {
		// 其他错误，返回错误信息
		return
	}
	// 读取 session_data 文件内容
	content, err := os.ReadFile(sessionDataPath)
	if err != nil {
		return
	}
	// 返回文件内容的字符串表示
	SessionData = string(content)
	return
}

func QrcodeLoginScan() bool {
	params := make(map[string]interface{})
	params["source"] = "main-fe-header"
	loginQrcodeGenerateRespData := ReqGet[LoginQrcodeGenerateRespData](WebQrcodeGenerate, params)
	qrUrl := loginQrcodeGenerateRespData.Data.Url
	qrcodeKey := loginQrcodeGenerateRespData.Data.QrcodeKey
	printQrcode(qrUrl)
	QrcodeLoginCallback(qrcodeKey)
	isVip := CheckBigVip()
	if isVip {
		err := writeSessionDataToLocalFile()
		if err != nil {
			fmt.Println("写入文件失败。。。。")
		}
	}
	return isVip
}

func printQrcode(data string) {
	// 生成二维码
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		fmt.Println("生成二维码时出错:", err)
		os.Exit(1)
	}
	// 使用字符串格式打印二维码到控制台
	fmt.Println(qr.ToSmallString(true))
}

func QrcodeLoginCallback(qrcodeKey string) {
	params := make(map[string]interface{})
	params["source"] = "main-fe-header"
	params["qrcode_key"] = qrcodeKey
	for {
		data := ReqGet[LoginCallbackRespData](WebQrcodePoll, params)
		// 86090 已扫码
		// 0 完成
		if data.Data.Code == 0 {
			getSessionDataFromUrl(data.Data.Url)
			break
		} else {
			continue
		}
	}
}

func getSessionDataFromUrl(dataUrl string) {
	// 定义正则表达式模式
	pattern := `SESSDATA=([^&]+)`
	// 编译正则表达式
	regex := regexp.MustCompile(pattern)
	// 使用正则表达式查找匹配的子串
	matches := regex.FindStringSubmatch(dataUrl)
	// 检查是否找到匹配的子串
	if len(matches) >= 2 {
		SessionData = matches[1]
	}
}

func writeSessionDataToLocalFile() error {
	// 获取用户的主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// 构建 session_data 文件的完整路径
	folderPath := filepath.Join(homeDir, ".bilibili-download")
	filePath := filepath.Join(folderPath, "session_data")
	// 检查文件夹是否存在，如果不存在则创建
	if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
		fmt.Println("无法创建文件夹:", err)
		return err
	}
	// 打开文件，如果文件不存在则自动创建
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return err
	}
	defer file.Close()

	// 覆盖写入字符串内容
	_, err = file.WriteString(SessionData)
	if err != nil {
		fmt.Println("写入文件内容失败:", err)
		return err
	}
	return nil
}

func CheckAccount() (flag bool, msg string) {
	if SessionData == "" {
		return false, "❌ 未登录账号"
	} else {
		vip := CheckBigVip()
		if vip {
			return vip, "✅ 大会员已登陆"
		} else {
			return !vip, "⭕️ 大会员已过期"
		}
	}
}

func CheckBigVip() bool {
	params := make(map[string]interface{})
	data := ReqGet[NavUserRespData](WebInterfaceNav, params)
	// 我的大会员类型是2，status是1;普通用户类型是1，status是0
	if data.Data.VipStatus >= 1 && data.Data.VipType >= 2 {
		return true
	}
	return false
}

func DownloadMp4Video(bvId, savePath string, qn int) {
	data := Mp4VideoPlay(bvId, qn)
	videoUrl := data.Data.Durl[0].Url
	if videoUrl != "" {
		filename := fmt.Sprintf("video_%v_%v%v", bvId, time.Now().Unix(), ".mp4")
		client := &http.Client{}
		request, err := http.NewRequest("GET", videoUrl, nil)
		if err != nil {
			log.Println(err)
			return
		}
		setUserAgent(request)
		SetCookie(request)
		request.Header.Set("Accept", "*/*")
		request.Header.Set("Accept-Language", "en-US,en;q=0.5")
		request.Header.Set("Accept-Encoding", "gzip, deflate, br")
		request.Header.Set("Range", "bytes=0-")                               // Range 的值要为 bytes=0- 才能下载完整视频
		request.Header.Set("Referer", "https://www.bilibili.com/video/"+bvId) // 必需添加
		request.Header.Set("Origin", "https://www.bilibili.com")
		request.Header.Set("Connection", "keep-alive")

		rsp, err := client.Do(request)
		if err != nil {
			log.Println(err)
			return
		}
		defer rsp.Body.Close()

		path := filepath.Join(savePath, filename)
		out, err := os.Create(path)
		if err != nil {
			log.Printf("err: %v", err)
			return
		}
		defer out.Close()
		dr := &Downloader{
			rsp.Body,
			rsp.ContentLength,
			0,
		}
		io.Copy(out, dr)
	}
	return
}

func DownloadVideo(bvId, savePath string, qn int) (filename string) {
	data := playerPlayUrl(bvId)
	video := data.Data.Dash.Video
	videoUrl := ""
	for _, stream := range video {
		if stream.ID == qn {
			videoUrl = stream.BaseURL
			break
		}
	}
	if videoUrl != "" {
		filename = fmt.Sprintf("video_%v_%v%v", bvId, time.Now().Unix(), ".m4s")
		client := &http.Client{}
		request, err := http.NewRequest("GET", videoUrl, nil)
		if err != nil {
			log.Println(err)
			return
		}
		setUserAgent(request)
		SetCookie(request)
		request.Header.Set("Accept", "*/*")
		request.Header.Set("Accept-Language", "en-US,en;q=0.5")
		request.Header.Set("Accept-Encoding", "gzip, deflate, br")
		request.Header.Set("Range", "bytes=0-")                               // Range 的值要为 bytes=0- 才能下载完整视频
		request.Header.Set("Referer", "https://www.bilibili.com/video/"+bvId) // 必需添加
		request.Header.Set("Origin", "https://www.bilibili.com")
		request.Header.Set("Connection", "keep-alive")

		rsp, err := client.Do(request)
		if err != nil {
			log.Println(err)
			return
		}
		defer rsp.Body.Close()

		path := filepath.Join(savePath, filename)
		out, err := os.Create(path)
		if err != nil {
			log.Printf("err: %v", err)
			return
		}
		defer out.Close()
		dr := &Downloader{
			rsp.Body,
			rsp.ContentLength,
			0,
		}
		io.Copy(out, dr)
	}
	return
}

func DownloadAudio(bvId, savePath string, qn int) (filename string) {
	data := playerPlayUrl(bvId)
	audio := data.Data.Dash.Audio
	audioUrl := ""
	for _, stream := range audio {
		if stream.ID == qn {
			audioUrl = stream.BaseURL
			break
		}
	}
	if audioUrl != "" {
		filename = fmt.Sprintf("audio_%v_%v%v", bvId, time.Now().Unix(), ".m4s")
		client := &http.Client{}
		request, err := http.NewRequest("GET", audioUrl, nil)
		if err != nil {
			log.Println(err)
			return
		}
		setUserAgent(request)
		SetCookie(request)
		request.Header.Set("Accept", "*/*")
		request.Header.Set("Accept-Language", "en-US,en;q=0.5")
		request.Header.Set("Accept-Encoding", "gzip, deflate, br")
		request.Header.Set("Range", "bytes=0-")                               // Range 的值要为 bytes=0- 才能下载完整视频
		request.Header.Set("Referer", "https://www.bilibili.com/video/"+bvId) // 必需添加
		request.Header.Set("Origin", "https://www.bilibili.com")
		request.Header.Set("Connection", "keep-alive")

		rsp, err := client.Do(request)
		if err != nil {
			log.Println(err)
			return
		}
		defer rsp.Body.Close()

		path := filepath.Join(savePath, filename)
		out, err := os.Create(path)
		if err != nil {
			log.Printf("err: %v", err)
			return
		}
		defer out.Close()
		dr := &Downloader{
			rsp.Body,
			rsp.ContentLength,
			0,
		}
		io.Copy(out, dr)
	}
	return
}

func HasSubtitle(bvid string) bool {
	response := webInterfaceView(bvid)
	if response != nil && len(response.Data.Subtitle.List) > 0 {
		return true
	}
	return false
}

func GetSubtitleInfo() {

}

func GetVideoQuality(bvid string) ([]string, []int) {
	data := playerPlayUrl(bvid)
	quality := data.Data.AcceptQuality
	description := data.Data.AcceptDescription
	return description, quality
}

func GetAudioQuality(bvid string) ([]string, []int) {
	data := playerPlayUrl(bvid)
	var quality []int
	var description []string
	for _, audio := range data.Data.Dash.Audio {
		quality = append(quality, audio.ID)
		switch audio.ID {
		case 30216:
			description = append(description, "64  kbps")
		case 30232:
			description = append(description, "128 kbps")
		case 30280:
			description = append(description, "320 kbps")
		}
	}
	return description, quality
}

// 获取视频播放信息
func playerPlayUrl(bvid string) (videoRespData *Response[VideoPlayRespData]) {
	webInterfaceViewRespData := webInterfaceView(bvid)
	params := make(map[string]interface{})
	params["fnval"] = 4048
	params["avid"] = webInterfaceViewRespData.Data.Aid
	params["cid"] = webInterfaceViewRespData.Data.Cid
	videoRespData = ReqGet[VideoPlayRespData](PlayerPlayUrl, params)
	return
}

// Mp4VideoPlay 获取视频播放信息
func Mp4VideoPlay(bvid string, qn int) (videoRespData *Response[Mp4VideoRespData]) {
	webInterfaceViewRespData := webInterfaceView(bvid)
	params := make(map[string]interface{})
	params["bvid"] = bvid
	params["cid"] = webInterfaceViewRespData.Data.Cid
	params["qn"] = qn
	videoRespData = ReqGet[Mp4VideoRespData](PlayerPlayUrl, params)
	return
}

// 获取视频页面信息
func webInterfaceView(bvid string) (webInterfaceViewRespData *Response[WebInterfaceViewRespData]) {
	params := make(map[string]interface{})
	params["bvid"] = bvid
	webInterfaceViewRespData = ReqGet[WebInterfaceViewRespData](WebInterfaceView, params)
	return
}

func ReqGet[T WebInterfaceViewRespData | VideoPlayRespData | LoginCallbackRespData | LoginQrcodeGenerateRespData | NavUserRespData | Mp4VideoRespData](reqUrl string, params map[string]interface{}) (videoRespData *Response[T]) {
	client := req.C().
		SetTimeout(5 * time.Second)
	if SessionData != "" {
		cookie := http.Cookie{Name: "SESSDATA", Value: SessionData, Expires: time.Now().Add(30 * 24 * 60 * 60 * time.Second)}
		client.SetCommonCookies(&cookie)
	}
	var errMsg Resp
	resp, err := client.R().
		SetQueryParamsAnyType(params).
		SetSuccessResult(&videoRespData). // Unmarshal response body into userInfo automatically if status code is between 200 and 299.
		SetErrorResult(&errMsg).          // Unmarshal response body into errMsg automatically if status code >= 400.
		EnableDump().                     // Enable dump at request level, only print dump content if there is an error or some unknown situation occurs to help troubleshoot.
		Get(reqUrl)

	if err != nil { // Error handling.
		log.Println("error:", err)
		log.Println("raw content:")
		log.Println(resp.Dump()) // Record raw content when error occurs.
		return
	}

	if resp.IsErrorState() { // Status code >= 400.
		fmt.Println(errMsg.Message) // Record error message returned.
		return
	}

	if resp.IsSuccessState() { // Status code is between 200 and 299.
		return
	}

	// Unknown status code.
	log.Println("unknown status", resp.Status)
	log.Println("raw content:")
	log.Println(resp.Dump()) // Record raw content when server returned unknown status code.
	return
}

func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
}
func SetCookie(req *http.Request) {
	cookie := http.Cookie{Name: "SESSDATA", Value: SessionData, Expires: time.Now().Add(30 * 24 * 60 * 60 * time.Second)}
	req.AddCookie(&cookie)
}

// RemoveFiles 删除文件
func RemoveFiles(fileList *[]string) error {
	for _, file := range *fileList {
		if err := os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

func GetSavePath() (savePath string, err error) {
	savePath, err = os.Getwd()
	savePath = strings.TrimSpace(savePath)
	return
}
