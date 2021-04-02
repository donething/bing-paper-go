// 获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	. "bing-paper-go/config"
	"bing-paper-go/icon"
	"bing-paper-go/logger"
	"bing-paper-go/models"
	"encoding/json"
	"fmt"
	"github.com/donething/utils-go/dofile"
	"github.com/donething/utils-go/dohttp"
	"github.com/getlantern/systray"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// 壁纸保存的路径
	fileNameTimeFormat = "20060102"

	host = "https://cn.bing.com"
	// idx: 为0表示当天，1表示昨天；n: 获取壁纸的数量
	papersURL = "https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=10&" +
		"pid=hp&FORM=BEHPTB&uhd=1&uhdwidth=3840&uhdheight=2160"
)

var (
	client = dohttp.New(30*time.Second, false, false)
)

func main() {
	// 显示托盘
	go func() {
		systray.Run(onReady, nil)
	}()

	// 下载
	logger.Info.Println("开始定时下载必应壁纸：")
	run()
	ticker := time.NewTicker(11 * time.Hour)
	for range ticker.C {
		run()
	}
}

// 下载
func run() {
	for !dohttp.CheckNetworkConn() {
		logger.Warn.Printf("不能连接网络，继续等待……")
		time.Sleep(1 * time.Minute)
	}
	// 保存壁纸
	//obtainAllPapers()

	err := obtainLatestPapers()
	if err != nil {
		logger.Error.Printf("下载最新壁纸时出错：%s\n", err)
	}
}

// 显示systray托盘
func onReady() {
	systray.SetIcon(icon.Tray)
	systray.SetTitle("下载每日Bing壁纸")
	systray.SetTooltip("下载每日Bing壁纸")

	mOpenPaperFold := systray.AddMenuItem("打开壁纸文件夹", "打开壁纸文件夹")
	mIsMissPapers := systray.AddMenuItem("检测缺失的壁纸", "检测缺失的壁纸")
	mOpenLog := systray.AddMenuItem("打开日志", "打开日志文件")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	for {
		select {
		case <-mOpenPaperFold.ClickedCh:
			err := dofile.OpenAs(Conf.Dir)
			if err != nil {
				logger.Error.Printf("打开路径(%s)出错：%s\n", Conf.Dir, err)
			}
		case <-mIsMissPapers.ClickedCh:
			checkMissingPapers()
		case <-mOpenLog.ClickedCh:
			err := dofile.OpenAs(logger.LogName)
			if err != nil {
				logger.Error.Printf("打开日志文件(%s)出错：%s\n", logger.LogName, err)
			}
		case <-mQuit.ClickedCh:
			// 退出程序
			logger.Info.Println("退出程序")
			systray.Quit()
			os.Exit(0)
		}
	}
}

// 获取必应壁纸
func obtainLatestPapers() error {
	// 获取必应壁纸的URL
	papersJSON, err := client.GetText(papersURL, nil)
	if err != nil {
		return fmt.Errorf("获取数据（%s）出错：%s\n", papersURL, err)
	}

	var ps models.BingPapers
	err = json.Unmarshal([]byte(papersJSON), &ps)
	if err != nil {
		return fmt.Errorf("解析json数据（%s）出错：%s\n", papersJSON, err)
	}

	// 保存壁纸为文件
	for _, p := range ps.Images {
		re := regexp.MustCompile(`id=(.+?)&`)
		// p.Startdate 的时间点比对应图片的信息晚一天，所以用 Enddate 代替
		name := p.Enddate + `_` + re.FindStringSubmatch(p.URL)[1]
		path := filepath.Join(Conf.Dir, name)
		exist, err := dofile.Exists(path)
		if err != nil {
			logger.Error.Printf("判断路径是否存在时出错：%s\n", err)
			continue
		}
		if exist {
			logger.Warn.Printf("本地已存在同名文件，结束此次获取壁纸：%s\n", path)
			break
		}

		_, err = client.Download(host+p.URL, path, true, nil)
		if err != nil {
			logger.Error.Printf("获取网络图片（%s）时保存文件（%s）出错：%s\n", host+p.URL, path, err)
			continue
		}

		logger.Info.Printf("图片（%s）保存完毕\n", path)
	}
	logger.Info.Println("本次图片保存完毕")
	return nil
}

func checkMissingPapers() {
	// 解析最先和最后的两个文件的时间
	// 这两个时间可能不是应该下载的壁纸的时间范围
	// 不过依然以这两个时间为准
	logger.Info.Println("开始检测缺失壁纸：")
	files, err := ioutil.ReadDir(Conf.Dir)
	if err != nil {
		logger.Error.Printf("读取目录(%s)出错：%s\n", Conf.Dir, err)
		return
	}
	if len(files) == 0 {
		return
	}

	start := files[0].Name()
	start = start[0:strings.Index(start, "_")]
	end := files[len(files)-1].Name()
	end = end[0:strings.Index(end, "_")]

	startDate, err := time.Parse(fileNameTimeFormat, start)
	if err != nil {
		logger.Error.Printf("解析时间出错：%s\n", err)
		return
	}
	endDate, err := time.Parse(fileNameTimeFormat, end)
	if err != nil {
		logger.Error.Printf("解析时间出错：%s\n", err)
		return
	}

	// 将所有壁纸的文件名合并为一个字符串，方便后面检测包含
	allPapersText := ""
	for _, f := range files {
		allPapersText += f.Name() + " "
	}

	// 开始判断
	index := 0
	for date := startDate; date.Before(endDate); date = date.Add(24 * time.Hour) {
		format := date.Format(fileNameTimeFormat)
		if !strings.Contains(allPapersText, format) {
			logger.Warn.Printf("此日壁纸不存在：%s\n", format)
		}
		index++
	}
	logger.Info.Println("检测缺失壁纸的操作已完成")
}
