package util

import (
	"io/ioutil"
	"net/http"
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
