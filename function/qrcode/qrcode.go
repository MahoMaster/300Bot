package qrcode

import goqrcode "github.com/skip2/go-qrcode"

func EncodeToPNG(text string, size int) ([]byte, error) {
	return goqrcode.Encode(text, goqrcode.Medium, size)
}

