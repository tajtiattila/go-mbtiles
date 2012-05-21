package main

// provide a simple display for our maps with modestmaps
// so an mbtiles file can be displayed without further dependencies

import (
	"html/template"
	"net/http"
)

var mmtmpl = template.Must(template.New("mmtmpl").Parse(mmtext))

func modestmaps(w http.ResponseWriter, req *http.Request) {
	metadata, err := MbtMetadata(db_conn)
	if err != nil {
		http.Error(w, "metadata query error: " + err.Error(), 500)
		return
	}
	err = mmtmpl.Execute(w, metadata)
	if err != nil {
		http.Error(w, "template error: " + err.Error(), 500)
	}
}

var mmtext = `<html>
	<head>
		<title>{{.Name}}</title>
		<script type="text/javascript"
			src="https://raw.github.com/stamen/modestmaps-js/master/modestmaps.min.js"></script>
		<script type="text/javascript">
			var MM = com.modestmaps;
			var map;
			function initMap() {
				var layer = new MM.Layer(new MM.MapProvider(function(coord) {
					var img = parseInt(coord.zoom) + '/' + parseInt(coord.column) + '/'+ parseInt(coord.row) + '.jpg';
					return '/tiles/' + img;
				}))
				map = new MM.Map('map', layer);
				map.setCenterZoom(new MM.Location({{.Center.Lon}}, {{.Center.Lat}}), {{.Center.Zoom}});
				map.setZoomRange({{.MinZoom}},{{.MaxZoom}});
			}
		</script>
		<style>
			body {margin: 0; }
			#map { width: 100%; height:100%; }
		</style>
	</head>
<body onload="initMap()">
<div id="map">
</div>
</body>
</html>
`
