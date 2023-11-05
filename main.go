package main

import (
	"fmt"
	"time"
)

func main() {
	x()
}

func x() {
	hasFfmpeg, err := FfmpegVersion()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("📺 BiliBili 视频下载! ")
	GetLocalSessionData()
	flag, msg := CheckAccount()
	fmt.Println(msg)
	if !flag {
		isLoginVip := AskIsLoginVip()
		if isLoginVip {
			loginScan := QrcodeLoginScan()
			if loginScan {
				flag = loginScan
				fmt.Println("👏 欢迎尊贵的大会员用户！👏")
			} else {
				fmt.Println("☹️ 抱歉您不是大会员用户！☹️")
			}
		}
	}
	// 输入视频链接或者BV
	bvId := AskBV()
	fmt.Println(bvId)
	// 保存路径
	savePath, err := GetSavePath()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("📁视频存储路径:", savePath)
	if !hasFfmpeg || !flag {
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
		filename := fmt.Sprintf("video_%v_%v_%v_%v%v", bvId, videoQuality, audioQuality, time.Now().Format("2006-01-02 15:04:05"), ".mp4")
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
