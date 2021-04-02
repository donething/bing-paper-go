package config

import (
	"bing-paper-go/logger"
	"bing-paper-go/models"
	"bing-paper-go/utils"
	"encoding/json"
	"github.com/donething/utils-go/dofile"
	"os"
	"path"
)

const (
	// 配置文件名
	Name = "bing-paper-go.json"
)

var (
	Conf     models.Config
	confPath string
)

// 初始化配置文件
func init() {
	confPath = path.Join(Name)
	exist, err := dofile.Exists(confPath)
	utils.Fatal(err)
	if exist {
		logger.Info.Printf("读取配置文件：%s\n", confPath)
		bs, err := dofile.Read(confPath)
		utils.Fatal(err)
		err = json.Unmarshal(bs, &Conf)
		utils.Fatal(err)
		//logger.Info.Printf("配置文件内容：%s\n", string(bs))
	} else {
		logger.Info.Printf("创建配置文件：%s\n", confPath)
		bs, err := json.MarshalIndent(Conf, "", "  ")
		utils.Fatal(err)
		_, err = dofile.Write(bs, confPath, os.O_CREATE, 0644)
		utils.Fatal(err)
	}
}
