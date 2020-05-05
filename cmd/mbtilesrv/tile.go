package main

import (
	"bytes"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"image"
	"image/color"
	"image/png"
)

var emptytile []byte

var nstfont *truetype.Font

func init() {
	im := image.NewRGBA(image.Rect(0, 0, tilesize, tilesize))
	var buf bytes.Buffer
	err := png.Encode(&buf, im)
	if err != nil {
		panic(err)
	}
	emptytile = buf.Bytes()

	nstfont, err = freetype.ParseFont(luxiSansFontData())
	if err != nil {
		panic(err)
	}
}

func nosuchtile(v ...interface{}) []byte {
	im := image.NewRGBA(image.Rect(0, 0, tilesize, tilesize))
	col := color.RGBA{255, 0, 0, 255}
	for i := 0; i < tilesize; i++ {
		im.Set(i, 0, col)
		im.Set(i, tilesize-1, col)
		im.Set(0, i, col)
		im.Set(tilesize-1, i, col)
	}
	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(nstfont)
	ctx.SetFontSize(16)
	ctx.SetClip(im.Bounds())
	ctx.SetDst(im)
	ctx.SetSrc(image.Black)
	for i, n := range v {
		_, err := ctx.DrawString(fmt.Sprint(n), freetype.Pt(30, 30+i*20))
		if err != nil {
			fmt.Println(err)
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, im)
	if err != nil {
		fmt.Println(err)
	}
	return buf.Bytes()
}
