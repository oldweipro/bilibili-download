package main

import (
	"fmt"
)

const (
	dajuyuan = "BV1GU4y1x7XT"
	full8k   = "BV1s34y1C7yo"
)

func main() {
	x()
}

func x() {
	fmt.Println("📺 BiliBili 视频下载! ")
	GetLocalSessionData()
	flag, msg := CheckAccount()
	fmt.Println(msg)
	// 输入视频链接或者BV
	bvId := AskBV()
	fmt.Println(bvId)
	// 保存路径
	savePath := AskSavePath()
	fmt.Println(savePath)
	// 下载类型
	downloadType := AskSelectDownloadType()
	for _, dType := range downloadType {
		switch dType {
		case "视频":
			// 依次选择保存的分辨率
			quality := AskSelectVideoQuality(bvId)
			// 是否需要登陆
			if quality > 80 {
				if !flag {
					QrcodeLoginScan()
				}
			}
			fmt.Println("开始下载")
			DownloadVideo(bvId, savePath, quality)
		case "音频":
			fmt.Println("音频下载")
		}
	}
	fmt.Println("\n✅ 下载完成!")
}
