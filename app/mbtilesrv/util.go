package main

import (
	"math"
)

func deg_rad(d float64) float64 {
	return d * math.Pi / 180
}

func rad_deg(r float64) float64 {
	return r * 180 / math.Pi
}

func xyz_lonlat(x, y, z int) (lon, lat float64) {
	n := 1 << uint(z)
	fact := tilesize * float64(n)
	lon = float64(x)*360/fact - 180
	prj := (1 - float64(y)*2/fact)*math.Pi
	lat = rad_deg(math.Atan(math.Sinh(prj)))
	return
}

func lonlatz_xy(lon, lat float64, z int) (x, y int) {
	n := 1 << uint(z)
	fact := tilesize * float64(n)
	x = int((lon + 180) * fact / 360)
	prj := math.Log(math.Tan(math.Pi/4 + deg_rad(lat/2)))
	y = int((1 - prj)/math.Pi*fact/2)
	return
}

