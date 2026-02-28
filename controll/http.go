package controll

import (
	"300Bot/function/chatGPT"
	"300Bot/function/immortal"
	"300Bot/function/qrcode"
	"300Bot/send"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Test(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Fprintf(w, "hello")
}

func LumaCode2Info(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	mode := r.FormValue("mode")
	// log.Println(code)
	// log.Println(mode)
	var res = make(map[string]interface{})
	res["code"] = 0
	info, err := immortal.Code2Info(code)
	if err != nil {
		res["msg"] = err.Error()
		res["code"] = 1
		res["data"] = nil
	} else {
		if mode == "1" || mode == "2" || mode == "3" {
			var infoMap = make(map[string]interface{})
			err = json.Unmarshal([]byte(info), &infoMap)
			if err != nil {
				res["msg"] = err.Error()
				res["code"] = 1
				res["data"] = nil
			} else {
				res["data"] = infoMap
			}
		}
	}
	resp, _ := json.Marshal(res)
	w.Write(resp)
}

func LumaReport(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	progress := r.FormValue("progress")
	log.Println(progress)
	mode := r.FormValue("mode")
	var res = make(map[string]interface{})
	res["code"] = 0

	immortal.BreakReport(code, progress, mode)
	resp, _ := json.Marshal(res)
	w.Write(resp)
}

func SendMeQQ(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	msg := r.FormValue("msg")
	send.SendPrivatePost(675559614, msg)
	var res = make(map[string]interface{})
	res["code"] = 0
	resp, _ := json.Marshal(res)
	w.Write(resp)
}

func JustChat(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var reqData = make(map[string]interface{})
	json.NewDecoder(r.Body).Decode(&reqData)
	msg := reqData["msg"].(string)
	qq := reqData["qq"].(string)

	resp, _ := chatGPT.JustChatGpt(msg, qq)

	var res = make(map[string]interface{})
	res["code"] = 0
	res["data"] = resp

	resData, _ := json.Marshal(res)

	w.Write(resData)
}

func SendQQMsg(w http.ResponseWriter, r *http.Request) {
	var res = make(map[string]interface{})
	res["code"] = 0

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		res["code"] = 1
		res["msg"] = "只支持POST"
		resp, _ := json.Marshal(res)
		w.Write(resp)
		return
	}

	qqStr := strings.TrimPrefix(r.URL.Path, "/sendQQMsg/")
	qqStr = strings.Trim(qqStr, "/")
	qq, err := strconv.ParseFloat(qqStr, 64)

	if err != nil {
		res["code"] = 1
		res["msg"] = "qq参数错误"
		resp, _ := json.Marshal(res)
		w.Write(resp)
		return
	}

	type sendQQMsgJSON struct {
		Title     string `json:"title"`
		Message   string `json:"message"`
		ImageFile string `json:"imageFile"`
		QRCode    string `json:"qrCode"`
	}

	var title string
	var message string
	var imageFileValue string
	var qrCodeText string

	contentType := r.Header.Get("Content-Type")
	contentType = strings.ToLower(contentType)

	if strings.HasPrefix(contentType, "application/json") {
		var req sendQQMsgJSON
		decodeErr := json.NewDecoder(r.Body).Decode(&req)
		if decodeErr != nil {
			res["code"] = 1
			res["msg"] = "json解析失败"
			resp, _ := json.Marshal(res)
			w.Write(resp)
			return
		}
		title = strings.TrimSpace(req.Title)
		message = strings.TrimSpace(req.Message)
		imageFileValue = strings.TrimSpace(req.ImageFile)
		qrCodeText = strings.TrimSpace(req.QRCode)
	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		parseErr := r.ParseMultipartForm(20 << 20)
		if parseErr != nil {
			res["code"] = 1
			res["msg"] = "form解析失败"
			resp, _ := json.Marshal(res)
			w.Write(resp)
			return
		}
		title = strings.TrimSpace(r.FormValue("title"))
		message = strings.TrimSpace(r.FormValue("message"))
		imageFileValue = strings.TrimSpace(r.FormValue("imageFile"))
		qrCodeText = strings.TrimSpace(r.FormValue("qrCode"))
	} else {
		r.ParseForm()
		title = strings.TrimSpace(r.FormValue("title"))
		message = strings.TrimSpace(r.FormValue("message"))
		imageFileValue = strings.TrimSpace(r.FormValue("imageFile"))
		qrCodeText = strings.TrimSpace(r.FormValue("qrCode"))
	}

	msgText := ""
	if title != "" && message != "" {
		msgText = title + "\n" + message
	} else if title != "" {
		msgText = title
	} else if message != "" {
		msgText = message
	}

	if msgText != "" {
		send.SendPrivatePost(qq, msgText)
	}

	sendTempImageFile := func(tmpPath string) {
		send.SendPrivateImageFile(qq, tmpPath)
		go func(p string) {
			time.Sleep(5 * time.Second)
			os.Remove(p)
		}(tmpPath)
	}

	sendExistingImageFile := func(absPath string) {
		send.SendPrivateImageFile(qq, absPath)
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		file, header, err := r.FormFile("imageFile")
		if err == nil {
			defer file.Close()

			_ = os.MkdirAll("./static/temp", 0755)
			ext := filepath.Ext(header.Filename)
			tmpFile, tmpErr := os.CreateTemp("./static/temp", "upload-*"+ext)
			if tmpErr == nil {
				tmpPath, _ := filepath.Abs(tmpFile.Name())
				_, copyErr := io.Copy(tmpFile, file)
				tmpFile.Close()
				if copyErr == nil {
					sendTempImageFile(tmpPath)
				} else {
					os.Remove(tmpPath)
				}
			}
		}
	}

	if imageFileValue != "" {
		if strings.HasPrefix(imageFileValue, "http://") || strings.HasPrefix(imageFileValue, "https://") {
			send.SendPrivatePost(qq, `[CQ:image,file=`+imageFileValue+`]`)
		} else if strings.HasPrefix(imageFileValue, "file:///") {
			send.SendPrivatePost(qq, `[CQ:image,file=`+imageFileValue+`]`)
		} else if strings.HasPrefix(imageFileValue, "data:") || strings.HasPrefix(imageFileValue, "base64:") || strings.HasPrefix(imageFileValue, "base64,") || strings.Contains(imageFileValue, "base64,") {
			b64 := ""
			ext := ".png"
			if strings.HasPrefix(imageFileValue, "data:") {
				idx := strings.Index(imageFileValue, "base64,")
				if idx >= 0 {
					meta := imageFileValue[:idx]
					if strings.Contains(meta, "image/jpeg") {
						ext = ".jpg"
					} else if strings.Contains(meta, "image/jpg") {
						ext = ".jpg"
					} else if strings.Contains(meta, "image/png") {
						ext = ".png"
					} else if strings.Contains(meta, "image/gif") {
						ext = ".gif"
					} else if strings.Contains(meta, "image/webp") {
						ext = ".webp"
					}
					b64 = strings.TrimSpace(imageFileValue[idx+7:])
				}
			} else if strings.HasPrefix(imageFileValue, "base64,") {
				b64 = strings.TrimSpace(strings.TrimPrefix(imageFileValue, "base64,"))
			} else if strings.HasPrefix(imageFileValue, "base64:") {
				b64 = strings.TrimSpace(strings.TrimPrefix(imageFileValue, "base64:"))
			} else {
				idx := strings.Index(imageFileValue, "base64,")
				if idx >= 0 {
					b64 = strings.TrimSpace(imageFileValue[idx+7:])
				}
			}

			if b64 != "" {
				imgBytes, decErr := base64.StdEncoding.DecodeString(b64)
				if decErr == nil {
					_ = os.MkdirAll("./static/temp", 0755)
					tmpFile, tmpErr := os.CreateTemp("./static/temp", "uploadb64-*"+ext)
					if tmpErr == nil {
						tmpPath, _ := filepath.Abs(tmpFile.Name())
						_, writeErr := tmpFile.Write(imgBytes)
						tmpFile.Close()
						if writeErr == nil {
							sendTempImageFile(tmpPath)
						} else {
							os.Remove(tmpPath)
						}
					}
				}
			}
		} else {
			p := imageFileValue
			if !filepath.IsAbs(p) {
				p, _ = filepath.Abs(p)
			}
			sendExistingImageFile(p)
		}
	}

	if qrCodeText != "" {
		_ = os.MkdirAll("./static/temp", 0755)
		pngBytes, qrErr := qrcode.EncodeToPNG(qrCodeText, 256)
		if qrErr == nil {
			tmpFile, tmpErr := os.CreateTemp("./static/temp", "qrcode-*.png")
			if tmpErr == nil {
				tmpPath, _ := filepath.Abs(tmpFile.Name())
				_, writeErr := tmpFile.Write(pngBytes)
				tmpFile.Close()
				if writeErr == nil {
					send.SendPrivateImageFile(qq, tmpPath)
				}
				go func(p string) {
					time.Sleep(5 * time.Second)
					os.Remove(p)
				}(tmpPath)
			}
		}
	}

	resp, _ := json.Marshal(res)
	w.Write(resp)
}
