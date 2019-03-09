// 定时获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	"github.com/donething/utils-go/dohttp"
	"testing"
	"time"
)

func Test_obtainAllPapers(t *testing.T) {
	obtainAllPapers()
}

func Test_checkFiles(t *testing.T) {
	checkFiles()
}

func Test_getPic(t *testing.T) {
	client := dohttp.New(180*time.Second, false, false)
	n, err := client.GetFile(`https://cn.bing.com/th?id=OHR.GrapeHarvest_ZH-CN9372743517_1920x1080.jpg&rf=NorthMale_1920x1080.jpg&pid=hp`, nil, `E:/Temp/bing.jpg`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("文件下载完成：%d\n", n)
}
