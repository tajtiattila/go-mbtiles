package main

import (
	"bytes"
	"github.com/tajtiattila/go-mbtiles/mbtiles"
	"html"
	"io/ioutil"
	"net/http"
	"time"
)

type MapboxjsTemplate struct {
	mbt         *mbtiles.Map
	debug       bool
	cachedtitle string
	data        []byte
}

func (m *MapboxjsTemplate) Execute(w http.ResponseWriter, r *http.Request) {
	n := m.mbt.Metadata().Name
	if n == "" {
		n = "MBTileSrv"
	}
	if n != m.cachedtitle || m.data == nil {
		m.cachedtitle = n
		txt := mapboxjstext
		if m.debug {
			data, err := ioutil.ReadFile("index.html")
			if err == nil {
				txt = data
			}
		}
		const sep1, sep2 = "<title>", "</title>"
		t1, t2 := bytes.Index(txt, []byte(sep1)), bytes.Index(txt, []byte(sep2))
		if t1 > 0 && t2 > 0 && bytes.IndexByte(txt[t1:t2], '<') < 0 {
			c := make([]byte, 0, len(txt)+len(n))
			c = append(c, txt[:t1+len(sep1)]...)
			c = append(c, html.EscapeString(n)...)
			c = append(c, txt[t2:]...)
			txt = c
		}
		m.data = txt
	}
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(m.data))
}

var mapboxjstext = []byte(`<html>
<head>
	<title>MBTileSrv</title>
	<link href='//api.tiles.mapbox.com/mapbox.js/v1.1.0/mapbox.css' rel='stylesheet' />
	<!--[if lte IE 8]>
		<link href='//api.tiles.mapbox.com/mapbox.js/v1.1.0/mapbox.ie.css' rel='stylesheet' />
	<![endif]-->
	<script src='//api.tiles.mapbox.com/mapbox.js/v1.1.0/mapbox.js'></script>

	<style>
		body { margin:0; }
		#map {
			width:100%;
			height:100%;
			background:#eee url(./images/bg.png);
		}
	</style>
</head>
<body>
	<div id='map' class='dark'></div>
	<script type='text/javascript'>
		var map = L.mapbox.map('map', './map.json');
		map.gridControl.options.follow = true;
		L.control.scale().addTo(map);
	</script>
</body>
</html>`)
