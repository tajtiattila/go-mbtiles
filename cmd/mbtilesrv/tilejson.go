package main

import (
	"bytes"
	"encoding/json"
	"github.com/tajtiattila/go-mbtiles/mbtiles"
	"io"
	"time"
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

func TileJson(mbt *mbtiles.Map, callback string) (io.ReadSeeker, time.Time, error) {
	md := mbt.Metadata()

	mapdata := &MapData{
		"1.0.0",
		md.Name,
		md.MinZoom,
		md.MaxZoom,
		[]float64{md.Bounds.W, md.Bounds.S, md.Bounds.E, md.Bounds.N},
		[]float64{md.Center.Lat, md.Center.Lon, md.Center.Zoom},
		[]string{"./tiles/{z}/{x}/{y}.png"},
		[]string{"./grids/{z}/{x}/{y}.json"},
		md.Template,
		md.Legend,
	}

	var buf bytes.Buffer
	if callback != "" {
		buf.WriteString(callback + "(")
	}
	if err := json.NewEncoder(&buf).Encode(mapdata); err != nil {
		return nil, time.Time{}, err
	}
	if callback != "" {
		buf.WriteString(");")
	}

	return bytes.NewReader(buf.Bytes()), mbt.Mtime, nil
}
