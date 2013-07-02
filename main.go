package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func chk_fatal(err error) {
	if err != nil {
		log.Fatal()
	}
}

var addr = flag.String("addr", ":10998", "http service address")
var markmissing = flag.Bool("markmissing", false, "mark missing tiles")

var modestmaps = flag.Bool("modestmaps", false, "serve modestmaps")
var leaflet = flag.String("leaflet", "", "serve leaflet with path to its dist folder")
var wax = flag.String("wax", "", "serve wax with path to its dist folder")
var serve = flag.String("serve", "", "additional paths to serve")

var tile_content_type string
var mbt *MBTiles

const tilesize = 256

func main() {
	flag.Parse()
	if *modestmaps && *leaflet != "" {
		log.Fatal("options -modestmaps and -leaflet are mutually exclusive")
	}

	if len(flag.Args()) != 1 {
		log.Fatal("exactly one .mbtiles file must be specified")
	}

	var err error
	mbt, err = OpenMBTiles(flag.Arg(0))
	chk_fatal(err)
	defer mbt.Close()
	mbt.AutoReload()

	servezxy("/tiles/", tiler)
	servezxy("/grids/", gridder)
	servefn("/map.json", "", func(req *http.Request) (io.ReadSeeker, error) {
		return TileJson(mbt, "") })
	servefn("/map.jsonp", "text/javascript", func(req *http.Request) (io.ReadSeeker, error) {
		return TileJson(mbt, req.URL.Query().Get("callback")) })

	if *modestmaps {
		enable_bgimg()
		enable_modestmaps(mbt)
	} else if *leaflet != "" {
		enable_bgimg()
		enable_leaflet(mbt, *leaflet)
	} else {
		enable_cache("/lib/")
		http.Handle("/", http.HandlerFunc(
			func(w http.ResponseWriter, req *http.Request) {
				fn := "index.html"
				rs, err := os.Open(fn)
				if err == nil {
					http.ServeContent(w, req, fn, time.Time{}, rs)
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
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
			log.Printf("serving: %s -> %s\n", mapping, source)
		}
	}
	err = http.ListenAndServe(*addr, nil)
	chk_fatal(err)
}

func tiler(w http.ResponseWriter, req *http.Request, z, x, y int) error {
	blob, err := mbt.GetTile(z, x, y)
	if err == ErrTileNotFound && *markmissing {
		log.Println("notile", z, x, y)
		blob, err = nosuchtile("no such tile", z, x, y), nil
	}
	if err == nil {
		http.ServeContent(w, req, "tile.png", mbt.Mtime, bytes.NewReader(blob))
	}
	return err
}

func gridder(w http.ResponseWriter, req *http.Request, z, x, y int) error {
	log.Println("gridder", req.URL)
	blob, err := mbt.GetGridData(z, x, y, req.URL.Query().Get("callback"))
	if err == nil {
		http.ServeContent(w, req, "grid.js", time.Time{}, bytes.NewReader(blob))
	}
	return err
}

func zxynotfound(err error, w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.Path, "not found:", err)
	http.Error(w, req.URL.Path+" not found", http.StatusNotFound)
}

func servezxy(prefix string, f func(w http.ResponseWriter, req *http.Request, z, x, y int) error) {
	http.Handle(prefix, http.StripPrefix(prefix, http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			err := ErrTileNotFound
			parts := strings.Split(req.URL.Path, "/")
			if len(parts) == 3 {
				n := strings.IndexAny(parts[2], ".")
				if n != -1 {
					parts[2] = parts[2][:n]
				}
				args := make([]int, 3)
				for i, s := range parts {
					args[i], err = strconv.Atoi(s)
					if err != nil {
						zxynotfound(err, w, req)
						return
					}
				}
				z, x, y := args[0], args[1], args[2]
				// Flip Y coordinate because MBTiles files are TMS
				y = (1 << uint(z)) - 1 - y
				err = f(w, req, z, x, y)
				if err == nil {
					return
				}
				if err != ErrTileNotFound {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			zxynotfound(err, w, req)
		})))
}

func servefn(pth string, ctyp string, f func(req *http.Request) (io.ReadSeeker, error)) {
	http.Handle(pth, http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			rs, err := f(req)
			if ctyp != "" {
				w.Header().Set("Content-Type", ctyp)
			}
			if err == nil {
				http.ServeContent(w, req, pth, time.Time{}, rs)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}))
}

