// 定时获取必应壁纸
// 参考：https://www.v2ex.com/t/157267
package main

import (
	"io/ioutil"
	"testing"
)

func Test_obtainAllPapers(t *testing.T) {
	obtainAllPapers()
}

func Test_checkFiles(t *testing.T) {
	checkFiles()
}

func Test_readBytes(t *testing.T) {
	bs, err := ioutil.ReadAll()
}
