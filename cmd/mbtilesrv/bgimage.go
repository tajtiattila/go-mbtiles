package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"time"
)

const (
	servepath = "/images/bg.png"
	bgsize    = 16
	bgmax     = bgsize - 1
)

var bgimg []byte

func init() {
	r := image.Rect(0, 0, bgsize, bgsize)
	im := image.NewRGBA(r)
	col := color.RGBA{240, 230, 188, 128}
	col2 := col
	col2.A = 66
	for i := 0; i < bgsize; i++ {
		im.Set(i, i, col)
		im.Set(bgmax-i, i, col)
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, im)
	if err != nil {
		panic(err)
	}
	bgimg = buf.Bytes()
}

func enable_bgimg() {
	http.Handle(servepath, http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			http.ServeContent(w, req, servepath,
				time.Time{}, bytes.NewReader(bgimg))
		}))
}
