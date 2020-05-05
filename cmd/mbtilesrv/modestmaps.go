package main

// provide a simple display for our maps with modestmaps
// so an mbtiles file can be displayed without further dependencies

import (
	"github.com/tajtiattila/go-mbtiles/mbtiles"
	"html/template"
	"net/http"
)

func enable_modestmaps(mbt *mbtiles.Map) error {
	mmtmpl, err := template.New("mmtmpl").Parse(mmtext)
	if err != nil {
		return err
	}
	http.Handle("/", http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			metadata := mbt.Metadata()
			err = mmtmpl.Execute(w, metadata)
			if err != nil {
				http.Error(w, "template error: "+err.Error(), 500)
			}
		}))
	return nil
}

var mmtext = `<html>
	<head>
		<title>{{.Name}}</title>
		<script type="text/javascript"
			src="//raw.github.com/stamen/modestmaps-js/master/modestmaps.min.js"></script>
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
