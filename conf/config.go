package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// config config
type config struct {
	Name string `json:"name"`

	Port string `json:"port"`
	//调用发送等api的端口
	ApiPort string `json:"apiPort"`
	//
}

const (
	// JwtKey 私钥
	SecretKey = ""
	// passWd 加盐
	// Salt = ""
)

var (
	Config config
)

func file_get_contents(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}
func init() {
	var content []byte
	// 读取配置文件
	content, err := file_get_contents("./conf/conf.json")
	if err != nil {
		fmt.Println("配置文件读取失败")
		panic(err.Error())
	}
	err = json.Unmarshal([]byte(content), &Config)
	if err != nil {
		fmt.Println("配置文件无效")
		panic(err.Error())
	}

	fmt.Println("run on ", Config.Port)
}
