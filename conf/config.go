package conf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// config config
type config struct {
	Name string `json:"name"`

	Port string `json:"port"`
	//调用发送等api的端口
	ApiPort string `json:"apiPort"`
	ApiUrl  string `json:"apiUrl"`
	Host    string `json:"host"`
	//机器人qq号
	BotQQ   string `json:"botQQ"`
	BotName string `json:"botName"`
	//最高权限QQ
	Manager string `json:"manager"`

	DatabaseHost     string `json:"databaseHost"`
	DatabaseUser     string `json:"databaseUser"`
	DatabasePassword string `json:"databasePassword"`
	BotDatabaseName  string `json:"botDatabaseName"`
	HeroDatabaseName string `json:"heroDatabaseName"`
	ImmortalbaseName string `json:"immortalbaseName"`

	ChatGPTKey    string `json:"chatGPTkey"`
	WetherApiCode string `json:"wetherApiCode"`
	VPN           string `json:"VPN"`

	MoneyList []string `json:"moneyList"` //赞助列表
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
	return io.ReadAll(f)
}
func init() {
	var content []byte
	// 读取配置文件
	content, err := file_get_contents("./conf/conf.local.json")
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
