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
)

var (
	host = `https://bing.com`
	// 将n设为3而不是1，是为了避免几天没打开电脑，而导致漏掉某天的壁纸
	papersURL = host + `/HPImageArchive.aspx?format=js&idx=0&n=3`
	client    = dohttp.New(180*time.Second, false, false)
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
	err := obtainLatestPapers()
	if err != nil {
		log.Fatal(err)
	}
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
		name := p.Startdate + `_` + p.URL[strings.LastIndex(p.URL, `/`)+1:]
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
	log.Println("本次所有图片保存完毕")
	return nil
}

// 获取网站上的所有壁纸
// 参考：https://github.com/benheart/BingGallery/blob/master/bing_gallery_crawler_new.py
func obtainAllPapers() {
	pageUrl := `https://bing.ioliu.cn/?p=%d`
	resolution := "1920x1080" // 图片分辨率
	log.Println("开始下载所有图片：")
	for i := 6; i >= 1; i-- {
		// 获取网页文本
		url := fmt.Sprintf(pageUrl, i)
		text, err := client.GetText(url, nil)
		if err != nil {
			log.Printf("获取网页（%s）文本出错：%s\n", url, err)
			continue
		}

		// 解析HTML
		dom, err := goquery.NewDocumentFromReader(strings.NewReader(text))
		if err != nil {
			log.Printf("解析HTML出错：%s\n", err)
			continue
		}

		// 得到壁纸真实URL
		dom.Find(".item").Each(func(i int, selection *goquery.Selection) {
			// 获取壁纸真实的URL
			src, has := selection.Find("img").Attr("src")
			if !has {
				log.Printf("没有找到img的src属性：%s\n", selection.Text())
				return
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
			calendar = calendar.Add(-24 * time.Hour)
			name := calendar.Format("20060102") + "_" + theUrl[strings.LastIndex(theUrl, "/")+1:]

			dst := filepath.Join(PapersPath, name)
			// 如果文件已存在，则取消下载
			exist, err := dofile.PathExists(dst)
			if err != nil {
				log.Printf("判断路径（%s）是否存在时出错：%s\n", dst, err)
				return
			}
			if exist {
				log.Printf("本地已存在同名文件（%s）\n", dst)
				return
			}

			// 保存到文件
			_, err = client.GetFile(theUrl, nil, filepath.Join(PapersPath, name))
			if err != nil {
				log.Printf("下载网络图片（%s） ==> （%s）出错：%s\n", theUrl, name, err)
				return
			}

			time.Sleep(2 * time.Second)
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
