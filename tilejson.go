package main

import (
	"bytes"
	"encoding/json"
	"io"
)

type MapData struct {
	TileJson string `json:"tilejson"`
	Name string `json:"name"`
	MinZoom int `json:"minzoom"`
	MaxZoom int `json:"maxzoom"`
	Bounds []float64 `json:"bounds"`
	Center []float64 `json:"center"`
	Tiles []string `json:"tiles"`
	Grids []string `json:"grids"`
	Template string `json:"template"`
	Legend string `json:"legend"`
}

func TileJson(mbtiles *MBTiles, callback string) (io.ReadSeeker, error) {
	md, err := mbtiles.Metadata()
	if err != nil {
		return nil, err
	}

	mapdata := &MapData{
		"1.0.0",
		md.Name,
		md.MinZoom,
		md.MaxZoom,
		[]float64{md.Bounds.W, md.Bounds.S, md.Bounds.E, md.Bounds.N},
		[]float64{md.Center.Lat, md.Center.Lon, md.Center.Zoom},
		[]string{"/tiles/{z}/{x}/{y}.png"},
		[]string{"/grids/{z}/{x}/{y}.json"},
		md.Template,
		md.Legend,
	}

	var buf bytes.Buffer
	if callback != "" {
		buf.WriteString(callback + "(")
	}
	if err = json.NewEncoder(&buf).Encode(mapdata); err != nil {
		return nil, err
	}
	if callback != "" {
		buf.WriteString(");")
	}

	return bytes.NewReader(buf.Bytes()), nil
}
