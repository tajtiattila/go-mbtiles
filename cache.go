package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type cacheitem struct {
	url      string
	data     []byte
	mtime    time.Time
	err      error // fetch error
}

var cache map[string]*cacheitem


func init() {
	paths := []string{
		"https://raw.github.com/mapbox/easey/gh-pages/src/easey.js",
		"https://raw.github.com/mapbox/easey/gh-pages/src/easey.handlers.js",
		"https://raw.github.com/mapbox/wax/master/ext/modestmaps.min.js",
		"https://raw.github.com/mapbox/wax/master/dist/wax.mm.js",
		"https://raw.github.com/mapbox/wax/master/theme/controls.css",
		"https://raw.github.com/mapbox/wax/master/theme/map-controls.png",
		"https://raw.github.com/mapbox/wax/master/theme/blank.gif",
	}
	cache = make(map[string]*cacheitem)
	for _, u := range paths {
		cache[path.Base(u)] = &cacheitem{url:u}
	}
}

func trycachedir(base string, e ...string) string {
	if base != "" {
		pth := path.Join(base, path.Join(e...))
		if _, err := os.Stat(pth); err == nil {
			return pth
		}
	}
	return ""
}

const (
	cachename  = "info.tajti.mbtilesrv"
)

type cachedir struct {
	base string
	path string
	full bool
}
var vcachedir = []cachedir{
	{"HOME", "Library/Caches", true},
	{"HOME", ".cache", false},
	{"USERPROFILE", ".cache", false},
	{"TEMP", "", false},
	{"TMP", "", false},
}

func sel(s1, s2 string, first bool) string {
	if first {
		return s1
	}
	return s2
}

func getcachedir(name string) string {
	short := name
	if n := strings.LastIndex(name, "."); n != -1 {
		short = name[n+1:]
	}
	for _, e := range vcachedir {
		base := os.Getenv(e.base)
		if base != "" {
			pth := path.Join(base, e.path, sel(name, short, e.full))
			if _, err := os.Stat(pth); err == nil {
				return pth
			}
		}
	}
	for _, e := range vcachedir {
		base := os.Getenv(e.base)
		if base != "" {
			pth := path.Join(base, e.path)
			if _, err := os.Stat(pth); err == nil {
				fullpth := path.Join(pth, sel(name, short, e.full))
				if err := os.Mkdir(fullpth, 0700); err == nil {
					return fullpth
				}
			}
		}
	}
	panic("no cache dir found")
}

func get_cached(fn string) (*cacheitem, error) {
	cached, has := cache[fn]
	if !has {
		return nil, os.ErrNotExist
	}

	if cached.err != nil {
		return nil, cached.err
	}
	if cached.data != nil {
		return cached, nil
	}

	lname := path.Join(getcachedir(cachename), fn)
	fi, ferr := os.Stat(lname)
	req, err := http.NewRequest("GET", cached.url, nil)
	if err != nil {
		cached.err = err
		return nil, err
	}
	if ferr == nil {
		req.Header.Add("If-Modified-Since", fi.ModTime().Format(http.TimeFormat))
	}
	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode == http.StatusNotModified || err != nil {
		if err != nil {
			// print error but use cached file
			fmt.Println("cache:", err)
		}
		data, err := ioutil.ReadFile(lname)
		if err != nil {
			return nil, err
		}
		cached.data = data
		cached.mtime = fi.ModTime()
		return cached, nil
	}
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	mt, terr := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if terr == nil {
		cached.mtime = mt
	}

	cached.data = data
	err = ioutil.WriteFile(lname, data, 0777)
	if err == nil {
		if terr == nil {
			os.Chtimes(lname, time.Now(), mt)
		}
	} else {
		fmt.Println("cache write error", err)
		os.Remove(lname) // try to remove in case of write error
	}
	if cached.data != nil {
		return cached, nil
	}
	return nil, err
}

func serve_cached(w http.ResponseWriter, req *http.Request) {
	cached, err := get_cached(req.URL.Path)
	if err == nil {
		http.ServeContent(w, req, cached.url, cached.mtime, bytes.NewReader(cached.data))
	} else {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

func enable_cache(pth string) {
	http.Handle(pth, http.StripPrefix(pth, http.HandlerFunc(serve_cached)))
}

