package main

import (
	"html/template"
	"net/http"
	"net/url"
)

type leafletparams struct {
	M       *Metadata
	Leaflet string
}

func enable_leaflet(mbtiles *MBTiles, libpath string) error {
	leaflettmpl, err := template.New("leaflettmpl").Parse(leaflettext)
	if err != nil {
		return err
	}
	liburl, err := url.Parse(libpath)
	if err != nil {
		return err
	}
	if !liburl.IsAbs() {
		// url is local path, serve contents at /leaflet/
		source := libpath
		libpath = "/leaflet/"
		http.Handle(libpath, http.StripPrefix(libpath,
			http.FileServer(http.Dir(source))))
	}
	http.Handle("/", http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			metadata, err := mbtiles.Metadata()
			if err != nil {
				http.Error(w, "metadata query error: "+err.Error(), 500)
				return
			}
			err = leaflettmpl.Execute(w, leafletparams{metadata, libpath})
			if err != nil {
				http.Error(w, "template error: "+err.Error(), 500)
			}
		}))
	return nil
}

var leaflettext = `<html>
	<head>
		<title>{{.M.Name}}</title>
		<link rel="stylesheet" href="{{.Leaflet}}/leaflet.css" />
		<!--[if lte IE 8]>
			<link rel="stylesheet" href="{{.Leaflet}}/leaflet.ie.css" />
		<![endif]-->
		<script src="{{.Leaflet}}/leaflet.js"></script>

		<script>
			var map;
			function initMap() {
				map = new L.Map('map', {
					center: new L.LatLng({{.M.Center.Lon}}, {{.M.Center.Lat}}),
					zoom: {{.M.Center.Zoom}}
				});
				var tmpl = '/tiles/{z}/{x}/{y}.png';
				var layer = new L.TileLayer(tmpl, {
					minZoom: {{.M.MinZoom}},
					maxZoom: {{.M.MaxZoom}}
				});
				var scale = new L.Control.Scale();
				map.addLayer(layer);
				map.addControl(scale);
			}
		</script>
		<style>
			body {margin: 0; }
			#map { width: 100%; height:100%;
				background:#eee url(/images/bg.png);
			}
		</style>
	</head>
<body onload="initMap()">
<div id="map">
</div>
</body>
</html>`
