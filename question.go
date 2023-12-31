package main

import (
	"github.com/AlecAivazis/survey/v2"
	"log"
	"os"
	"regexp"
	"strings"
)

func AskBV() string {
	for {
		input := ""
		prompt := &survey.Input{
			Message: "🔗请输入视频链接:",
		}
		err := survey.AskOne(prompt, &input)
		if err != nil {
			continue
		}
		// 构建正则表达式
		re := regexp.MustCompile(`\b(BV\w+)\b`)

		// 查找匹配的结果
		match := re.FindStringSubmatch(input)

		// 提取视频 ID
		if len(match) > 1 {
			input = match[1]
			return strings.TrimSpace(input)
		}
		continue
	}
}

func AskSavePath() string {
	for {
		inputPath := ""
		prompt := &survey.Input{
			Message: "📁请输入视频存储路径(如果为空, 默认为当前路径):",
		}
		err := survey.AskOne(prompt, &inputPath)
		if err != nil {
			continue
		}
		if inputPath == "" || inputPath == "\r\n" || inputPath == "\n" {
			inputPath, err = os.Getwd()
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		inputPath = strings.TrimSpace(inputPath)
		return inputPath
	}
}

// AskSelectDownloadType 下载类型
func AskSelectDownloadType() []string {
	var selectedOptions []string
	question := &survey.MultiSelect{
		Message: "请选择您下载的类型 (使用空格键进行多选):",
		Options: []string{"视频"},
	}
	err := survey.AskOne(question, &selectedOptions, survey.WithKeepFilter(true))
	if err != nil {
		return []string{"视频"}
	}
	if len(selectedOptions) == 0 {
		return []string{"视频"}
	}
	return selectedOptions
}

func AskIsLoginVip() bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: "是否登录大会员账号,以下载更高清晰度?",
	}
	survey.AskOne(prompt, &confirm)
	return confirm
}

// AskSelectMp4VideoQuality 选择视频清晰度
func AskSelectMp4VideoQuality(bv string) int {
	vq := ""
	data := Mp4VideoPlay(bv, 16)
	quality := data.Data.AcceptDescription
	nums := data.Data.AcceptQuality
	prompt := &survey.Select{
		Message: "请选择视频清晰度:",
		Options: quality,
	}
	err := survey.AskOne(prompt, &vq)
	if err != nil {
		return 16
	}
	videoQuality := findIntByQuality(quality, nums, vq)
	return videoQuality
}

// AskSelectVideoQuality 选择视频清晰度
func AskSelectVideoQuality(bv string) int {
	vq := ""
	quality, nums := GetVideoQuality(bv)
	prompt := &survey.Select{
		Message: "请选择视频清晰度:",
		Options: quality,
	}
	err := survey.AskOne(prompt, &vq)
	if err != nil {
		return 16
	}
	videoQuality := findIntByQuality(quality, nums, vq)
	return videoQuality
}

// AskSelectAudioQuality 选择音频清晰度
func AskSelectAudioQuality(bv string) int {
	vq := ""
	quality, nums := GetAudioQuality(bv)
	prompt := &survey.Select{
		Message: "请选择音频清晰度:",
		Options: quality,
	}
	err := survey.AskOne(prompt, &vq)
	if err != nil {
		return 30216
	}
	byQuality := findIntByQuality(quality, nums, vq)
	return byQuality
}

func findIntByQuality(quality []string, nums []int, targetQuality string) int {
	for i, q := range quality {
		if q == targetQuality {
			if i < len(nums) {
				return nums[i]
			}
			return 16
		}
	}
	return 16
}
