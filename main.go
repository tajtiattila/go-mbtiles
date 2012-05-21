package main

import (
	"bytes"
	"code.google.com/p/gosqlite/sqlite"
	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/truetype"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func chk_fatal(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var addr = flag.String("addr", ":10998", "http service address")
var markmissing = flag.Bool("markmissing", false, "mark missing tiles")

var tile_content_type string
var db_modtime time.Time
var db_conn *sqlite.Conn

var db_metadata_json []byte
var emptytile []byte

const tilesize = 256
const metadata_name = "metadata.json"

func main() {
	flag.Parse()
	dbname := flag.Arg(0)
	fi, err := os.Stat(dbname)
	chk_fatal(err)
	db_modtime = fi.ModTime()
	db_modtime = time.Time{}

	db_conn, err = sqlite.Open(dbname)
	chk_fatal(err)
	defer db_conn.Close()

	stmt, err := db_conn.Prepare("select name,value from metadata")
	chk_fatal(err)
	err = stmt.Exec()
	chk_fatal(err)

	metadata := make(map[string]string)
	for stmt.Next() {
		var name, value string
		err = stmt.Scan(&name, &value)
		chk_fatal(err)
		valjson := value
		switch name {
		case "bounds":
			valjson = fields(value, "w", "s", "e", "n")
		case "center":
			valjson = fields(value, "x", "y", "z")
		case "minzoom", "maxzoom":
			// nothing to do
		default:
			valjson = `"` + stringescape(value) + `"`
		}
		jsonline := sep + `"` + name + `":` + valjson
		sep = ",\n"
		metadata.WriteString(jsonline)
	}
	metadata.WriteString("\n}")
	db_metadata_json = metadata.Bytes()

	im := image.NewRGBA(image.Rect(0,0,tilesize,tilesize))
	var buf bytes.Buffer
	err = png.Encode(&buf, im)
	chk_fatal(err)
	emptytile = buf.Bytes()

	http.Handle("/tiles/", http.StripPrefix("/tiles/", http.HandlerFunc(tiler)))
	http.Handle("/" + metadata_name, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, metadata_name, db_modtime, bytes.NewReader(db_metadata_json))
	}))
	err = http.ListenAndServe(*addr, nil)
	chk_fatal(err)
}

func fields(s string, names ...string) string {
	parts := strings.Split(s, ",")
	res, sep := "{", ""
	for i, name := range names {
		res += sep + `"` + name + `":` + parts[i]
		sep = ","
	}
	return res + "}"
}

func stringescape(s string) string {
	res := ""
	for {
		n := strings.IndexAny(s, `\"`)
		if n == -1 {
			return res + s
		}
		part, esc := s[:n], `\` + s[n:n+1]
		s = s[n:]
		res += part + `\` + esc
	}
	return "" // not reached
}

var nstfont *truetype.Font

func nosuchtile(v ...interface{}) []byte {
	if nstfont == nil {
		data, err := ioutil.ReadFile("luxisr.ttf")
		if err != nil {
			fmt.Println(err)
			return nil
		}
		nstfont, err = freetype.ParseFont(data)
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}
	im := image.NewRGBA(image.Rect(0,0,tilesize,tilesize))
	col := color.RGBA{tilesize-1,0,0,tilesize-1}
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
		_, err := ctx.DrawString(fmt.Sprint(n), freetype.Pt(30, 30 + i*20))
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

func tiler(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) == 3 {
		n := strings.IndexAny(parts[2], ".")
		if n != -1 {
			parts[2] = parts[2][:n]
		}
		args := make([]int, 3)
		for i, s := range parts {
			var err error
			args[i], err = strconv.Atoi(s)
			if err != nil {
				fmt.Println("Bad request", parts)
				http.Error(w, "Bad request", 404)
				return
			}
		}
		args[2] = (1<<uint(args[0]))-1 - args[2]
		stmt, err := db_conn.Prepare(`select tile_data from tiles
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
		var blob []byte
		if err == nil {
			err = stmt.Exec(args[0], args[1], args[2])
			if err == nil && stmt.Next() {
				err = stmt.Scan(&blob)
			}
		}
		if blob == nil {
			fmt.Println("notile", args[0], args[1], args[2])
			if *markmissing {
				blob = nosuchtile("no such tile", args[0], args[1], args[2])
			} else {
				blob = emptytile
			}
		}
		content := bytes.NewReader(blob)
		http.ServeContent(w, req, "", db_modtime, content)
		return
	}
	fmt.Println(req.URL.Path)
	http.Error(w, req.URL.Path+" not found", 500)
}
