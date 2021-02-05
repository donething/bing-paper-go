// 获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	"bing-paper-go/icon"
	"bing-paper-go/models"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/donething/utils-go/dofile"
	"github.com/donething/utils-go/dohttp"
	"github.com/donething/utils-go/dolog"
	"github.com/getlantern/systray"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// 壁纸保存的路径
	PapersPath         = `C:/Do/MyData/Image/Bing`
	fileNameTimeFormat = "20060102"

	logName = "run.log"

	host = `https://cn.bing.com`
	// 将n设为3而不是1，是为了避免几天没打开电脑，而导致漏掉某天的壁纸
	papersURL    = `https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=3`
	allPapersURL = `https://bing.ioliu.cn/?p=%d`
	// 壁纸查漏：http://www.bingwallpaperhd.com
)

var (
	logFile *os.File
	client  = dohttp.New(180*time.Second, false, false)
)

func init() {
	// 保存日志到文件
	var err error
	logFile, err = dolog.LogToFile(logName, dofile.OAppend, dolog.LogFormat)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	defer logFile.Close()
	// 显示托盘
	go func() {
		systray.Run(onReady, nil)
	}()

	// 下载
	log.Println("开始定时下载必应壁纸：")
	run()
	ticker := time.NewTicker(12 * time.Hour)
	for range ticker.C {
		run()
	}
}

// 下载
func run() {
	for !dohttp.CheckNetworkConn() {
		time.Sleep(1 * time.Minute)
	}
	// 保存壁纸
	//obtainAllPapers()
	err := obtainLatestPapers()
	if err != nil {
		log.Printf("下载最新壁纸时出错：%s\n", err)
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
			err := dofile.OpenAs(PapersPath)
			if err != nil {
				log.Printf("打开路径(%s)出错：%s\n", PapersPath, err)
			}
		case <-mIsMissPapers.ClickedCh:
			checkMissingPapers()
		case <-mOpenLog.ClickedCh:
			err := dofile.OpenAs(logName)
			if err != nil {
				log.Printf("打开日志文件(%s)出错：%s\n", logName, err)
			}
		case <-mQuit.ClickedCh:
			// 退出程序
			log.Println("退出程序")
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
		name := p.Enddate + `_` + p.URL[strings.LastIndex(p.URL, `/`)+1:]
		path := filepath.Join(PapersPath, name)
		exist, err := dofile.Exists(path)
		if err != nil {
			log.Printf("判断路径是否存在时出错：%s\n", err)
			continue
		}
		if exist {
			// log.Printf("本地已存在同名文件（%s）\n", path)
			continue
		}

		_, err = client.Download(host+p.URL, path, true, nil)
		if err != nil {
			log.Printf("获取网络图片（%s）时保存文件（%s）出错：%s\n", host+p.URL, path, err)
			continue
		}

		// 检测图片完整性
		ok, err := dofile.CheckIntegrity(path)
		if err != nil {
			log.Printf("检测图片完整性时出错：%s\n", path)
			continue
		}
		if !ok {
			log.Printf("文件不完整：%s\n", path)
		}
		log.Printf("图片（%s）保存完毕\n", path)
	}
	log.Println("本日图片保存完毕")
	return nil
}

// 获取网站上的所有壁纸
// 参考：https://github.com/benheart/BingGallery/blob/master/bing_gallery_crawler_new.py
func obtainAllPapers() {
	resolution := "1920x1080" // 图片分辨率
	log.Println("开始下载所有图片：")
	var allHasDownload = false // 是否所有壁纸都已下载
	for i := 1; ; i++ {
		// 所有壁纸下载完毕后，退出
		if allHasDownload {
			break
		}
		// 获取网页文本
		url := fmt.Sprintf(allPapersURL, i)
		text, err := client.GetText(url, nil)
		if err != nil {
			log.Printf("获取网页（%s）文本出错：%s\n", url, err)
			continue
		}

		// 如果当前页和前一页的页数相同，则说明已读取完所有页数
		if strings.Contains(text, fmt.Sprintf(`<a href="/?p=%d">`, i)) {
			break
		}

		// 解析HTML
		dom, err := goquery.NewDocumentFromReader(strings.NewReader(text))
		if err != nil {
			log.Printf("解析HTML出错：%s\n", err)
			continue
		}

		// 得到壁纸真实URL
		// EachWithBreak()中的函数返回true时继续Each，返回false则退出Each
		dom.Find(".item").EachWithBreak(func(i int, selection *goquery.Selection) bool {
			// 获取壁纸真实的URL
			src, has := selection.Find("img").Attr("src")
			if !has {
				log.Printf("没有找到img的src属性：%s\n", selection.Text())
				return true
			}
			// theUrl格式：http://h1.ioliu.cn/bing/AbstractSaltBeds_ZH-CN8351691359_1920x1080.jpg
			theUrl := src[0:strings.LastIndex(src, "_")] + "_" + resolution + ".jpg"

			// 提取文件名
			theTime := selection.Find(".calendar").Text()
			calendar, err := time.Parse("2006-01-02", theTime)
			if err != nil {
				log.Printf("解析时间出错：%s\n", err)
				return true
			}
			name := calendar.Format(fileNameTimeFormat) + "_" + theUrl[strings.LastIndex(theUrl, "/")+1:]

			dst := filepath.Join(PapersPath, name)
			// 如果文件已存在，则取消下载
			exist, err := dofile.Exists(dst)
			if err != nil {
				log.Printf("判断路径（%s）是否存在时出错：%s\n", dst, err)
				return true
			}
			if exist {
				// 存在同名文件，则说明至今的壁纸都已获取完毕。准备退出for循环退出
				log.Printf("本地已存在同名文件（%s）\n", dst)
				allHasDownload = true
				return false

			}

			// 保存到文件
			_, err = client.Download(theUrl, dst, true, nil)
			if err != nil {
				log.Printf("下载网络图片（%s） ==> （%s）出错：%s\n", theUrl, name, err)
				return true
			}

			ok, err := dofile.CheckIntegrity(dst)
			if err != nil {
				log.Printf("检测文件（%s）的完整性出错：%s\n", dst, err)
				return true
			}
			if !ok {
				log.Printf("检测到下载的文件(%s)不完整\n", dst)
			}

			time.Sleep(1 * time.Second)
			return true
		})
	}
	log.Println("所有图片处理完毕")
}

// jpg文件完整性检测
func checkFiles() {
	paths, err := ioutil.ReadDir(PapersPath)
	if err != nil {
		log.Printf("读取目录（%s）出错：%s\n", PapersPath, err)
		return
	}

	for _, p := range paths {
		if p.IsDir() {
			continue
		}
		path := filepath.Join(PapersPath, p.Name())
		ok, err := dofile.CheckIntegrity(path)
		if err != nil {
			log.Printf("检测文件（%s）的完整性出错：%s\n", path, err)
			continue
		}
		if !ok {
			log.Printf("文件不完整：%s\n", path)
		}
	}
}

func checkMissingPapers() {
	// 解析最先和最后的两个文件的时间
	// 这两个时间可能不是应该下载的壁纸的时间范围
	// 不过依然以这两个时间为准
	log.Println("开始检测缺失壁纸：")
	files, err := ioutil.ReadDir(PapersPath)
	if err != nil {
		log.Printf("读取目录(%s)出错：%s\n", PapersPath, err)
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
		log.Printf("解析时间出错：%s\n", err)
		return
	}
	endDate, err := time.Parse(fileNameTimeFormat, end)
	if err != nil {
		log.Printf("解析时间出错：%s\n", err)
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
			log.Printf("此日壁纸不存在：%s\n", format)
		}
		index++
	}
	log.Println("检测缺失壁纸的操作已完成")
}
