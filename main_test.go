// 定时获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	"github.com/donething/utils-go/dohttp"
	"testing"
	"time"
)

func Test_getPic(t *testing.T) {
	client := dohttp.New(180*time.Second, false, false)
	n, err := client.Download(`https://cn.bing.com/th?id=OHR.GrapeHarvest_ZH-CN9372743517_1920x1080.jpg&rf=NorthMale_1920x1080.jpg&pid=hp`, `D:/Temp/bing.jpg`, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("文件下载完成：%d\n", n)
}

func Test_obtainLatestPapers(t *testing.T) {
	err := obtainLatestPapers()
	if err != nil {
		t.Fatal(err)
	}
}
