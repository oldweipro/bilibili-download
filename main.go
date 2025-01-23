package main

import (
	"bytes"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/imroc/req/v3"
	"github.com/skip2/go-qrcode"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var bvId = "BV1pqwFevEQ4"

func main() {
	if bvId == "" {
		return
	}
	err := FfmpegVersion()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("📺 BiliBili 视频下载! ")
	GetLocalSessionData()
	flag, msg := CheckAccount()
	fmt.Println(msg)
	if !flag {
		loginScan := QrcodeLoginScan()
		if loginScan {
			flag = loginScan
			fmt.Println("👏 欢迎尊贵的大会员用户！👏")
		} else {
			fmt.Println("☹️ 抱歉您不是大会员用户！☹️")
		}
	}
	fmt.Println(bvId)
	// 保存路径
	savePath, err := GetSavePath()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("📁视频存储路径:", savePath)
	if !flag {
		videoQuality := AskSelectMp4VideoQuality(bvId)
		DownloadMp4Video(bvId, savePath, videoQuality)
		return
	}
	// 依次选择保存的分辨率
	videoQuality := AskSelectVideoQuality(bvId)
	audioQuality := AskSelectAudioQuality(bvId)
	fmt.Println("开始下载")
	if videoQuality == 80 || videoQuality == 16 {
		DownloadMp4Video(bvId, savePath, videoQuality)
	} else {
		fmt.Println("⏱️ 请耐心等待视频下载 🎬")
		videoFile := DownloadVideo(bvId, savePath, videoQuality)
		fmt.Println("\n⏱️ 请耐心等待音频下载 🎵")
		audioFile := DownloadAudio(bvId, savePath, audioQuality)
		filename := fmt.Sprintf("video_%v_%v_%v_%v%v", bvId, videoQuality, audioQuality, time.Now().Unix(), ".mp4")
		fileList := []string{videoFile, audioFile}
		err = FfmpegMergeFile(&fileList, &filename)
		if err != nil {
			fmt.Println(err.Error())
		}
		err = RemoveFiles(&fileList)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("\n✅ 下载完成!")
}

//============================== bubbletea start ==============================

func AskSelectMp4VideoQuality(bv string) int {
	data := Mp4VideoPlay(bv, 16)
	quality := data.Data.AcceptDescription
	nums := data.Data.AcceptQuality

	m := model{
		qualityOptions:  quality,
		selectedQuality: quality[0], // 默认选择第一个
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("启动程序时出错: %v\n", err)
		return 16
	}

	// 返回所选质量对应的数字
	videoQuality := findIntByQuality(quality, nums, m.selectedQuality)
	return videoQuality
}

type model struct {
	qualityOptions  []string
	selectedQuality string
}

func (m model) Init() tea.Cmd {
	return nil
}

// 处理用户输入
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.selectedQuality = m.previousQuality()
		case "down":
			m.selectedQuality = m.nextQuality()
		case "enter":
			fmt.Printf("你选择的视频质量是: %s\n", m.selectedQuality)
			return m, tea.Quit
		}
	}
	return m, nil
}

// 渲染界面
func (m model) View() string {
	var sb strings.Builder
	sb.WriteString("请选择视频清晰度:\n")
	for _, quality := range m.qualityOptions {
		if quality == m.selectedQuality {
			sb.WriteString(fmt.Sprintf("> %s\n", quality))
		} else {
			sb.WriteString(fmt.Sprintf("  %s\n", quality))
		}
	}
	sb.WriteString("\n按上下键选择，按回车确认")
	return sb.String()
}

// 获取下一个质量
func (m model) nextQuality() string {
	for i, quality := range m.qualityOptions {
		if quality == m.selectedQuality {
			if i+1 < len(m.qualityOptions) {
				return m.qualityOptions[i+1]
			}
			return m.qualityOptions[0] // 循环回到第一个
		}
	}
	return m.qualityOptions[0] // 默认返回第一个
}

// 获取上一个质量
func (m model) previousQuality() string {
	for i, quality := range m.qualityOptions {
		if quality == m.selectedQuality {
			if i-1 >= 0 {
				return m.qualityOptions[i-1]
			}
			return m.qualityOptions[len(m.qualityOptions)-1] // 循环回到最后一个
		}
	}
	return m.qualityOptions[0] // 默认返回第一个
}

// 根据给定的质量列表和所选质量返回视频质量的数字表示
func findIntByQuality(quality []string, nums []int, selectedQuality string) int {
	for i, q := range quality {
		if q == selectedQuality {
			return nums[i]
		}
	}
	return nums[0] // 默认返回第一个质量
}
func AskSelectVideoQuality(bv string) int {
	quality, nums := GetVideoQuality(bv)

	m := model{
		qualityOptions:  quality,
		selectedQuality: quality[0], // 默认选择第一个
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("启动程序时出错: %v\n", err)
		return 16
	}

	// 返回所选质量对应的数字
	videoQuality := findIntByQuality(quality, nums, m.selectedQuality)
	return videoQuality
}

func AskSelectAudioQuality(bv string) int {
	quality, nums := GetAudioQuality(bv)

	m := model{
		qualityOptions:  quality,
		selectedQuality: quality[0], // 默认选择第一个
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("启动程序时出错: %v\n", err)
		return 16
	}

	// 返回所选质量对应的数字
	videoQuality := findIntByQuality(quality, nums, m.selectedQuality)
	return videoQuality
}

//============================== bubbletea end ==============================

// ============================= handle start ==============================

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

// ============================= handle end ==============================

//============================== ffmpeg start ==============================

// FfmpegVersion 检查是否安装ffmpeg
func FfmpegVersion() error {
	cmd := exec.Command("ffmpeg", "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return errors.New("未找到ffmpeg, 请先安装")
	}
	return nil
}

// FfmpegMergeFile 使用ffmpeg合并文件
func FfmpegMergeFile(fileList *[]string, outFile *string) error {
	var arg []string
	for _, fp := range *fileList {
		arg = append(arg, "-i", fp)
	}

	arg = append(arg, "-vcodec", "copy", "-acodec", "copy", *outFile)
	cmd := exec.Command("ffmpeg", arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: %s", "文件合并失败", out.String()))
	}
	return nil
}

//============================= ffmpeg end ==============================
