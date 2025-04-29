package util

import (
	"300Bot/conf"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	defer func() {
		if info := recover(); info != nil {
			fmt.Println("芜锁胃", info)
		}
	}()
	resp, err := http.Get(url)
	if err != nil {
		// panic(err)
		log.Println(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

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
		// panic(err)
		log.Println(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	return result
}

func ChatGPTHttpPost(urll string, data interface{}) []byte {
	defer func() {
		if info := recover(); info != nil {
			fmt.Println("芜锁胃", info)
		}
	}()
	proxyUrl, err := url.Parse("http://127.0.0.1:7078")
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// 超时时间：60秒
	client := &http.Client{Timeout: 60 * time.Second, Transport: transport}
	// js,err:=json.MarshalIndent(data,"","\t")
	jsonStr, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", urll, bytes.NewBuffer(jsonStr))
	if err != nil {
		// panic(err)
		log.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", conf.Config.ChatGPTKey))

	resp, err := client.Do(req)
	// resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		// panic(err)
		log.Println(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(result))
	return result
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
