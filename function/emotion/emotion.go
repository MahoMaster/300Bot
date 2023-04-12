package emotion

import (
	"300Bot/model"
	"300Bot/send"
	"300Bot/util"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
)

func SetImgBackground(id string, msg map[string]interface{}) {
	qq := msg["user_id"].(float64)
	flag := model.SetImgBackground(qq, id)
	if flag {
		send.SendGroupPost(msg["group_id"].(float64), `已修改`)
	}
}

func Synthesis(text string, msg map[string]interface{}) {
	qq := msg["user_id"].(float64)
	info := model.GetImgBackGroundInfo(qq)
	// fmt.Println(utf8.RuneCountInString(text))
	if utf8.RuneCountInString(text) > info.Max_length {
		send.SendGroupPost(msg["group_id"].(float64), `这样子的，不行啦`)
		return
	}
	name := "./static/temp/" + util.RandStr(5) + ".jpg"
	file, _ := os.Create(name)

	defer file.Close()
	back, _ := os.Open("./static/imgBackground/" + info.File_name)
	defer back.Close()
	bg, _, err := image.Decode(back)
	if err != nil {
		fmt.Println(err)
	}

	jpg := image.NewNRGBA(image.Rect(0, 0, info.Rect_x, info.Rect_y))

	var (
		dpi      float64 = 72                              // "screen resolution in Dots Per Inch"
		fontfile string  = "./static/ttf/" + info.Ttf_name // "filename of the ttf font"
		hinting  string  = "none"                          // "none | full"
		size     float64 = info.Font_size                  //"font size in points"
		spacing  float64 = info.Font_spacing               // "line spacing (e.g. 2 means double spaced)"
	)

	flag.Parse()
	fontBytes, err := os.ReadFile(fontfile)
	if err != nil {
		fmt.Println(err)
		return
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	fg := image.Black
	draw.Draw(jpg, jpg.Bounds(), bg, image.Point{}, draw.Src)
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(f)
	c.SetFontSize(size)
	c.SetClip(jpg.Bounds())
	c.SetDst(jpg)
	c.SetSrc(fg)

	switch hinting {
	default:
		c.SetHinting(font.HintingNone)
	case "full":
		c.SetHinting(font.HintingFull)
	}

	as := (float64(info.X_end-info.X_start) - float64(utf8.RuneCountInString(text))*size*spacing) / 2
	pt := freetype.Pt(info.X_start, info.Y_start)
	pt.X += c.PointToFixed(as)
	_, err = c.DrawString(text, pt)
	if err != nil {
		fmt.Println(err)
		return
	}

	png.Encode(file, jpg)

	path, _ := filepath.Abs(name)
	send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]`)
	go func() {
		time.Sleep(5 * time.Second)
		os.Remove(name)
	}()

}
