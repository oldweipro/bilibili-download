package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

var QnMap = map[string]struct {
	QN         int
	NeedCookie bool
	Detail     string
}{
	"4K":      {120, true, "超清 4K"},
	"1080P60": {116, true, "高清1080P60"},
	"1080P+":  {112, true, "高清1080P+"},
	"1080P":   {80, true, "1080P"},
	"720P60":  {74, true, "高清720P60"},
	"720P":    {64, false, "高清720P"},
	"480P":    {32, false, "清晰480P"},
	"360P":    {16, false, "流畅360P"},
}

type BilibiliCid struct {
	Bvid     string
	Cid      string
	Title    string
	QN       int
	PlayURLs []string
}

type Downloader struct {
	io.Reader
	Total   int64
	Current int64
}

func (d *Downloader) Read(p []byte) (n int, err error) {
	n, err = d.Reader.Read(p)

	d.Current += int64(n)

	fmt.Printf("\rprogress: %.2f%%", float64(d.Current)*100.0/float64(d.Total))

	return
}

func (c *BilibiliCid) Download(dir string) {
	for _, URL := range c.PlayURLs {
		u, err := url.Parse(URL)
		if err != nil {
			log.Printf("ERR: %v", err)
			continue
		}

		name := fmt.Sprintf("%v_%v_%v", c.Bvid, c.Title, path.Base(path.Base(u.Path)))
		//log.Printf("Downloading[%d]: name:%v\n\turl:%v\n", i, name, URL)

		client := &http.Client{}
		req, err := http.NewRequest("GET", URL, nil)
		if err != nil {
			log.Println(err)
			return
		}
		setUserAgent(req)
		SetCookie(req)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Range", "bytes=0-")                               // Range 的值要为 bytes=0- 才能下载完整视频
		req.Header.Set("Referer", "https://www.bilibili.com/video/"+BvId) // 必需添加
		req.Header.Set("Origin", "https://www.bilibili.com")
		req.Header.Set("Connection", "keep-alive")

		rsp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		defer rsp.Body.Close()

		path := filepath.Join(dir, name)
		//log.Printf("save to: %v", path)
		out, err := os.Create(path)
		if err != nil {
			log.Printf("err: %v", err)
			continue
		}
		defer out.Close()

		dr := &Downloader{
			rsp.Body,
			rsp.ContentLength,
			0,
		}

		io.Copy(out, dr)
		//fmt.Println("")
	}
}

func (c *BilibiliCid) getPlayURLs() {
	videoUrl := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%v&cid=%v&qn=%v&fourk=1", c.Bvid, c.Cid, c.QN)
	//fmt.Println(url)

	pl := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Quality int `json:"quality"`
			Durl    []struct {
				Order     int    `json:"order"`
				URL       string `json:"url"`
				BackupURL string `json:"backup_url"`
			} `json:"durl"`
		} `json:"data"`
	}{}

	data := rawGetURL(videoUrl, SetCookie)
	//fmt.Println(data)
	json.Unmarshal([]byte(data), &pl)

	for _, p := range pl.Data.Durl {
		//log.Printf("PlayList[%d]: quality %v order %v url %v %v", i, pl.Data.Quality, p.Order, p.URL, p.BackupURL)
		c.PlayURLs = append(c.PlayURLs, p.URL)
	}
}

func GetCidList(bvid string, qn int) []BilibiliCid {
	cl := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			Cid       int64  `json:"cid"`
			Page      int    `json:"page"`
			Part      string `json:"part"`
			Duration  int    `json:"duration"`
			Vid       string `json:"vid"`
			Dimension struct {
				Width  int `json:"width"`
				Height int `json:"height"`
				Rotate int `json:"rotate"`
			} `json:"Dimension"`
		} `json:"data"`
	}{}

	data := getURL("https://api.bilibili.com/x/player/pagelist?bvid=" + bvid)

	json.Unmarshal([]byte(data), &cl)

	if len(cl.Data) == 0 {
		log.Printf("ERR: get cid list failed")
	}

	var cids []BilibiliCid
	for _, d := range cl.Data {
		c := BilibiliCid{}
		c.Cid = strconv.FormatInt(d.Cid, 10)
		c.Title = d.Part
		c.Bvid = bvid
		c.QN = qn
		c.getPlayURLs()
		cids = append(cids, c)
		//log.Printf("CidList[%d]: %v %v %v %v", i, d.Cid, d.Part, d.Dimension.Width, d.Dimension.Height)
	}

	return cids
}

func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
}
func SetCookie(req *http.Request) {
	cookie := http.Cookie{Name: "SESSDATA", Value: SessionData, Expires: time.Now().Add(30 * 24 * 60 * 60 * time.Second)}
	//log.Printf("got bilibili cookie, SESSDATA:%v", SessionData)
	req.AddCookie(&cookie)
}

func getURL(url string) string {
	return rawGetURL(url, nil)
}

func rawGetURL(url string, headerSet func(*http.Request)) (s string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return
	}

	setUserAgent(req)

	if headerSet != nil {
		headerSet(req)
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("http return %v\n", res.StatusCode)
		return
	}

	rsp, _ := io.ReadAll(res.Body)

	s = string(rsp)

	return
}
