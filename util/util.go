package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// func HttpPost() 	{
//     resp, err := http.PostForm("http://www.01happy.com/demo/accept.php",
//         url.Values{"key": {"Value"}, "id": {"123"}})

//     if err != nil {
//         // handle error
//     }

//     defer resp.Body.Close()
//     body, err := ioutil.ReadAll(resp.Body)
//     if err != nil {
//         // handle error
//     }

//     fmt.Println(string(body))
// }

func HttpGet(url string) []byte {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	return body
}

func HttpPost(url string, data interface{}) []byte {
	defer func() {
		if info := recover(); info != nil {
			fmt.Println("芜锁胃", info)
		}
	}()
	// 超时时间：60秒
	client := &http.Client{Timeout: 60 * time.Second}
	jsonStr, _ := json.Marshal(data)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	return result
}

func DeletePreAndSufSpace(str string) string {
	strList := []byte(str)
	spaceCount, count := 0, len(strList)
	for i := 0; i <= len(strList)-1; i++ {
		if strList[i] == 32 {
			spaceCount++
		} else {
			break
		}
	}

	strList = strList[spaceCount:]
	spaceCount, count = 0, len(strList)
	for i := count - 1; i >= 0; i-- {
		if strList[i] == 32 {
			spaceCount++
		} else {
			break
		}
	}

	return string(strList[:count-spaceCount])
}

func Time2Str(t int64) string {
	timeTemplate1 := "2006-01-02 15:04:05" //常规类型
	// timeTemplate2 := "2006/01/02 15:04:05" //其他类型
	// timeTemplate3 := "2006-01-02"          //其他类型
	// timeTemplate4 := "15:04:05"            //其他类型
	// ======= 将时间戳格式化为日期字符串 =======
	return time.Unix(t, 0).Format(timeTemplate1) //输出：2019-01-08 13:50:30
	// log.Println(time.Unix(t, 0).Format(timeTemplate2)) //输出：2019/01/08 13:50:30
	// log.Println(time.Unix(t, 0).Format(timeTemplate3)) //输出：2019-01-08
	// log.Println(time.Unix(t, 0).Format(timeTemplate4)) //输出：13:50:30
}
