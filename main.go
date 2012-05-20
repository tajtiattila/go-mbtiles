package main

import (
	"bytes"
	"code.google.com/p/gosqlite/sqlite"
	"flag"
	"fmt"
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
var tile_content_type string
var db_modtime time.Time
var db_conn *sqlite.Conn

func main() {
	flag.Parse()
	dbname := flag.Arg(0)
	fi, err := os.Stat(dbname)
	chk_fatal(err)
	db_modtime = fi.ModTime()

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
		metadata[name] = value
		fmt.Println(name)
	}

	http.Handle("/tiles/", http.HandlerFunc(tiler))
	err = http.ListenAndServe(*addr, nil)
	chk_fatal(err)
}

func tiler(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/tiles/") {
		parts := strings.Split(req.URL.Path[7:], "/")
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
					http.Error(w, "Bad request", 404)
					return
				}
			}
			stmt, err := db_conn.Prepare(`select tile_data from tiles
 where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
			if err == nil {
				err = stmt.Exec(args[0], args[1], args[2])
				if err == nil {
					if !stmt.Next() {
						http.Error(w, req.URL.Path+" no such tile", 404)
					}
					var blob []byte
					err = stmt.Scan(&blob)
					if err == nil {
						content := bytes.NewReader(blob)
						http.ServeContent(w, req, "", db_modtime, content)
						return
					}
				}
			}
			http.Error(w, "hopp!" + err.Error(), 404)
		}
	}
	http.Error(w, req.URL.Path+" not found", 500)
}
