package main

import (
	"bufio"
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	BvId        string
	Page        int // 视频合集的分p, 如果不指定，默认为0表示全下
	Dir         string
	Qn          int
	SessionData string
)

func init() {
	app := &cli.App{
		Version: "1.0",
		Name:    "bilibili-download",
		Usage:   "命令行中下载 bilibili 视频",
		Action: func(c *cli.Context) error {
			ColorsPrintBF("📺 BiliBili 视频下载! ", 44, 33, true, true)
			return nil
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "bv",
			Aliases:     []string{"b"},
			Usage:       "视频的bv号",
			Destination: &BvId,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "dir",
			Aliases:     []string{"d"},
			Usage:       "视频存储位置(默认为当前路径)",
			Destination: &Dir,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "sessionData",
			Aliases:     []string{"s"},
			Usage:       "下载1080P以上清晰度的视频",
			Destination: &SessionData,
			Required:    false,
		},
		&cli.IntFlag{
			Name:        "page",
			Aliases:     []string{"p"},
			Usage:       "视频合集的分p, 如果不指定，默认为0表示全下",
			Destination: &Page,
			Required:    false,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// ColorsPrintF 打印带有前景色的文本。30黑色,31红色,32绿色,33黄色,34蓝色,35紫色,36深绿,37白色
func ColorsPrintF(message string, fg uint8, highlight bool, isNewLine bool) {
	end := ""
	hl := 0

	if highlight {
		hl = 1
	}
	if isNewLine {
		end = "\n"
	}

	fmt.Printf("\x1b[%d;%dm%s\x1b[0m%s", hl, fg, message, end)
}

// ColorsPrintBF 打印带有背景色和前景色的文本。40黑色,41红色,42绿色,43黄色,44蓝色,45紫色,46深绿,47白色
func ColorsPrintBF(message string, bg, fg uint8, highlight bool, isNewLine bool) {
	end := ""
	hl := 0

	if highlight {
		hl = 1
	}
	if isNewLine {
		end = "\n"
	}

	fmt.Printf("\x1b[%d;%d;%dm%s\x1b[0m%s", hl, bg, fg, message, end)
}

func InitArguments() {
	for {
		if BvId != "" {
			if match, err := regexp.MatchString("[B|b][V|v][0-9a-zA-Z]{10}\\b", BvId); err == nil && match {
				break
			} else {
				ColorsPrintF("BV号错误!", 31, false, true)
			}

		}

		reader := bufio.NewReader(os.Stdin)
		ColorsPrintF("? ", 32, false, false)
		ColorsPrintF("请输入视频BV号: ", 37, false, false)
		bv, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err.Error())
		}
		BvId = strings.TrimSpace(bv)
	}

	//for {
	//	if SessionData != "" {
	//		_qn := strings.ToUpper("1080P60")
	//		if v, ok := QnMap[_qn]; ok {
	//			Qn = v.QN
	//
	//			if v.NeedCookie && SessionData == "" {
	//				log.Fatalf("need set value of bilibili cookie[\"SESSDATA\"] when you download %v", v.Detail)
	//			}
	//			break
	//		} else {
	//			log.Fatalf("invalid qn value")
	//		}
	//	}
	//
	//	reader := bufio.NewReader(os.Stdin)
	//	ColorsPrintF("? ", 32, false, false)
	//	ColorsPrintF("请输入sessionData以下载更高清晰度: ", 37, false, false)
	//	session, err := reader.ReadString('\n')
	//	if err != nil {
	//		log.Fatal(err.Error())
	//	}
	//	SessionData = strings.TrimSpace(session)
	//}

	for {
		if Dir != "" {
			if fileInfo, err := os.Stat(Dir); err == nil && fileInfo.IsDir() {
				break
			} else {
				ColorsPrintF("路径错误!", 31, false, true)
			}
		}

		reader := bufio.NewReader(os.Stdin)
		ColorsPrintF("? ", 32, false, false)
		ColorsPrintF("请输入视频存储路径(如果为空, 默认为当前路径): ", 37, false, false)
		path, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err.Error())
		}
		if path == "" || path == "\r\n" || path == "\n" {
			path, err = os.Getwd()
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		Dir = strings.TrimSpace(path)
	}

	videoInfo, err := GetVideoInfo(BvId)
	if err != nil {
		return
	}
	if videoInfo.Aid == 0 {
		log.Fatal("未找到视频!")
		return
	}
	BVID = videoInfo.Bvid
	videoUrl, err := GetVideoUrl(
		fmt.Sprintf(
			"%sx/player/playurl?fnval=4048&avid=%d&cid=%d",
			BASE_URL,
			videoInfo.Aid,
			videoInfo.Cid,
		),
	)
	if err != nil {
		return
	}
	videoIndex, audioIndex := SelectQuality(
		videoUrl.Dash.GetVideoQualitys(),
		videoUrl.Dash.GetAudioQualitys(),
	)
	fmt.Println("videoIndex:", videoIndex)
	fmt.Println("audioIndex:", audioIndex)
	atoi, err := strconv.Atoi(videoIndex)
	videos := GetCidList(BvId, atoi)
	for i, v := range videos {
		if Page == 0 || Page == (i+1) {
			v.Download(Dir)
		}
	}
}
