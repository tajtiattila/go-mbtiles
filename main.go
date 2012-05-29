package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

func chk_fatal(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var addr = flag.String("addr", ":10998", "http service address")
var markmissing = flag.Bool("markmissing", false, "mark missing tiles")

var modestmaps = flag.Bool("modestmaps", false, "serve modestmaps")
var leaflet = flag.String("leaflet", "", "serve leaflet with path to its dist folder")
var serve = flag.String("serve", "", "additional paths to serve")

var tile_content_type string
var mbt *MBTiles

var db_metadata *Metadata
var db_metadata_json []byte

const tilesize = 256
const metadata_name = "metadata.json"

func main() {
	flag.Parse()
	if *modestmaps && *leaflet != "" {
		fmt.Println("options -modestmaps and -leaflet are mutually exclusive")
		os.Exit(1)
	}

	var err error
	mbt, err = OpenMBTiles(flag.Arg(0))
	chk_fatal(err)
	defer mbt.Close()
	mbt.AutoReload()

	db_metadata, err = mbt.Metadata()
	chk_fatal(err)

	db_metadata_json, err = json.Marshal(db_metadata)
	chk_fatal(err)

	http.Handle("/tiles/", http.StripPrefix("/tiles/", http.HandlerFunc(tiler)))
	http.Handle("/"+metadata_name, http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			http.ServeContent(w, req, metadata_name,
				mbt.Mtime, bytes.NewReader(db_metadata_json))
		}))
	if *modestmaps {
		enable_bgimg()
		enable_modestmaps()
	} else if *leaflet != "" {
		enable_bgimg()
		enable_leaflet(*leaflet)
	}

	if *serve != "" {
		for _, entry := range strings.Split(*serve, ",") {
			v := strings.SplitN(entry, ":", 2)
			var mapping, source string
			if len(v) == 2 {
				mapping, source = v[0], v[1]
			} else {
				mapping, source = path.Base(entry), entry
			}
			if mapping[0] != '/' {
				mapping = "/" + mapping
			}
			if mapping[len(mapping)-1] != '/' {
				mapping = mapping + "/"
			}
			http.Handle(mapping, http.StripPrefix(mapping, http.FileServer(http.Dir(source))))
			fmt.Printf("serving: %s -> %s\n", mapping, source)
		}
	}
	err = http.ListenAndServe(*addr, nil)
	chk_fatal(err)
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
				http.Error(w, "Bad request", http.StatusNotFound)
				return
			}
		}
		z, x, y := args[0], args[1], args[2]
		// Flip Y coordinate because MBTiles files are TMS
		y = (1 << uint(z)) - 1 - y
		blob, err := mbt.GetTile(z, x, y)
		if blob == nil {
			if err != ErrTileNotFound {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Println("notile", z, x, y)
			if *markmissing {
				blob = nosuchtile("no such tile", z, x, y)
			} else {
				blob = emptytile
			}
		}
		content := bytes.NewReader(blob)
		http.ServeContent(w, req, "", mbt.Mtime, content)
		return
	}
	fmt.Println(req.URL.Path)
	http.Error(w, req.URL.Path+" not found", http.StatusInternalServerError)
}
