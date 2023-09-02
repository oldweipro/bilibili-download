package main

import (
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"io"
	"log"
	"net/http"
	"strconv"
)

type ResultInfo[D VideoUrl | VideoInfo] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    D      `json:"data"`
}

type VideoInfo struct {
	Bvid  string `json:"bvid"`
	Aid   int    `json:"aid"`
	Cid   int    `json:"cid"`
	Title string `json:"title"`
}

type VideoUrl struct {
	Dash Dash `json:"dash"`
}

type Dash struct {
	Duration int     `json:"duration"`
	Videos   []Video `json:"video"`
	Audios   []Audio `json:"audio"`
}

type Video struct {
	Id        int    `json:"id"`
	BaseUrl   string `json:"baseUrl"`
	BandWidth int    `json:"bandwidth"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Audio struct {
	Id      int    `json:"id"`
	BaseUrl string `json:"baseUrl"`
}

// 获取所有视频质量文字描述
func (d *Dash) GetVideoQualitys() []string {
	var qualitys []string
	for _, v := range d.Videos {
		qualitys = append(qualitys, strconv.Itoa(v.Id))
	}

	return qualitys
}

// 获取所有音频质量的文字描述
func (d *Dash) GetAudioQualitys() []string {
	var qualitys []string
	for _, a := range d.Audios {
		qualitys = append(qualitys, strconv.Itoa(a.Id))
	}

	return qualitys
}

// 获取视频质量的文字描述
//func (v *Video) getQuality() string {
//	return fmt.Sprintf("%s | %dx%d | BandWidth: %d", "qua", v.Width, v.Height, v.BandWidth)
//}
//
//// 获取音频质量的文字描述
//func (a *Audio) getQuality() string {
//	return "qua"
//}

var (
	BASE_URL string = "https://api.bilibili.com/"
	BVID     string
	CLIENT   *http.Client = &http.Client{}
)

// 获取视频信息
func GetVideoInfo(bvid string) (VideoInfo, error) {
	var resultInfo ResultInfo[VideoInfo]
	url := fmt.Sprintf("%sx/web-interface/view?bvid=%s", BASE_URL, bvid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resultInfo.Data, err
	}
	SetCookie(req)
	resp, err := CLIENT.Do(req)
	if err != nil {
		return resultInfo.Data, err
	}
	defer resp.Body.Close()

	bodyString, err := ReadCloserToString(&resp.Body)
	if err != nil {
		return resultInfo.Data, err
	}

	err = json.Unmarshal([]byte(bodyString), &resultInfo)
	if err != nil {
		return resultInfo.Data, err
	}

	return resultInfo.Data, nil
}

// 获取视频下载连接地址等信息
func GetVideoUrl(url string) (VideoUrl, error) {
	var resultInfo ResultInfo[VideoUrl]
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resultInfo.Data, err
	}

	SetCookie(req)
	resp, err := CLIENT.Do(req)
	if err != nil {
		return resultInfo.Data, err
	}
	defer resp.Body.Close()

	bodyString, err := ReadCloserToString(&resp.Body)
	if err != nil {
		return resultInfo.Data, err
	}

	err = json.Unmarshal([]byte(bodyString), &resultInfo)
	if err != nil {
		return resultInfo.Data, err
	}

	return resultInfo.Data, nil
}

// 选择视频，音频质量，视频保存格式
func SelectQuality(videoQualitys, audioQualitys []string) (videoIndex, audioIndex string) {
	var qs = []*survey.Question{
		{
			Name: "VideoQuality",
			Prompt: &survey.Select{
				Message:  "选择视频画质: ",
				Options:  videoQualitys,
				VimMode:  true,
				PageSize: 10,
			},
		},
		{
			Name: "AudioQuality",
			Prompt: &survey.Select{
				Message: "选择音频质量: ",
				Options: audioQualitys,
				VimMode: true,
			},
		},
	}

	answers := struct {
		VideoQuality string
		AudioQuality string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		log.Fatal(err.Error())
	}
	return answers.VideoQuality, answers.AudioQuality
}
func ReadCloserToString(rc *io.ReadCloser) (string, error) {
	body, err := io.ReadAll(*rc)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
