package main

import (
	"bytes"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
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
