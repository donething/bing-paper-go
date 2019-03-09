// 获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	"donething/bing-paper-go/icon"
	"donething/bing-paper-go/models"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/donething/utils-go/dofile"
	"github.com/donething/utils-go/dohttp"
	"github.com/getlantern/systray"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// 壁纸保存的路径
	PapersPath = `D:/MyData/Image/Bing`
	logName    = "run.log"

	host = `https://cn.bing.com`
	// 将n设为3而不是1，是为了避免几天没打开电脑，而导致漏掉某天的壁纸
	papersURL    = `https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=3`
	allPapersURL = `https://bing.ioliu.cn/?p=%d`
)

var (
	client = dohttp.New(180*time.Second, false, false)
)

func init() {
	// 打印log时显示时间戳
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 将日志输出到屏幕和日志文件
	lf, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("打开日志文件出错：", err)
	}
	// 此句不能有，否则日志不能保存到文件中
	// defer lf.Close()
	// MultiWriter()的参数顺序也重要，如果使用"-H windowsgui"参数build，并且需要将日志保存到文件，
	// 则需要将日志文件的指针（lf）放到os.Stdout之前，否则log不会产生输出
	log.SetOutput(io.MultiWriter(lf, os.Stdout))
}

func main() {
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
	obtainAllPapers()
}

// 显示systray托盘
func onReady() {
	systray.SetIcon(icon.Tray)
	systray.SetTitle("下载每日Bing壁纸")
	systray.SetTooltip("下载每日Bing壁纸")

	mOpenPaperFold := systray.AddMenuItem("打开壁纸文件夹", "打开壁纸文件夹")
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
		exist, err := dofile.PathExists(path)
		if err != nil {
			log.Printf("判断路径（%s）是否存在时出错：%s\n", path, err)
			continue
		}
		if exist {
			// log.Printf("本地已存在同名文件（%s）\n", path)
			continue
		}

		_, err = client.GetFile(host+p.URL, nil, path)
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
			timeItems := strings.Split(theTime, "-")
			year, _ := strconv.Atoi(timeItems[0])
			month, _ := strconv.Atoi(timeItems[1])
			day, _ := strconv.Atoi(timeItems[2])
			calendar := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			name := calendar.Format("20060102") + "_" + theUrl[strings.LastIndex(theUrl, "/")+1:]

			dst := filepath.Join(PapersPath, name)
			// 如果文件已存在，则取消下载
			exist, err := dofile.PathExists(dst)
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
			_, err = client.GetFile(theUrl, nil, filepath.Join(PapersPath, name))
			if err != nil {
				log.Printf("下载网络图片（%s） ==> （%s）出错：%s\n", theUrl, name, err)
				return true
			}

			time.Sleep(1 * time.Second)
			return true
		})
	}
	log.Println("所有图片处理完毕")

	// 检测图片完整性
	log.Println("开始检测文件的完整性：")
	checkFiles()
	log.Println("所有文件检测完成")
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
