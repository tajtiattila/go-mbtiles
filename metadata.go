package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

type MbtBounds struct {
	N, S, E, W float64
}
type MbtCenter struct {
	Lon, Lat, Zoom float64
}
type Metadata struct {
	Bounds MbtBounds
	Center MbtCenter
	MinZoom, MaxZoom int
	Name, Description, Attribution, Legend, Template, Version string
	Errors []error
}

func MbtMetadata(conn *sqlite.Conn) (*Metadata, error) {
	stmt, err := conn.Prepare("select name,value from metadata")
	if err != nil {
		return nil, err
	}
	err = stmt.Exec()
	if err != nil {
		return nil, err
	}

	md := new(Metadata)
	sep := "{\n"
	for stmt.Next() {
		var name, value string
		err = stmt.Scan(&name, &value)
		if err != nil {
			return nil, err
		}
		var ve []error
		switch name {
		case "bounds":
			ve = fill(value, &md.Bounds.W, &md.Bounds.S, &md.Bounds.E, &md.Bounds.N)
		case "center":
			ve = fill(value, &md.Center.Lat, &md.Center.Lon, &md.Center.Zoom)
		case "minzoom":
			ve = fill(value, &md.MinZoom)
		case "maxzoom":
			ve = fill(value, &md.MaxZoom)
		default:
			if rv := reflect.ValueOf(&md).FieldByName(strings.Title(name)); rv.Kind() == reflect.String {
				rv.SetString(value)
			}
		}
		if ve != nil {
			md.Errors = append(md.Errors, ve...)
		}
	}
	return md, nil
}

func fill(s string, v ...interface{}) []error {
	ve := make([]error,0,len(v))
	parts := strings.Split(s, ",")
	for i := range v {
		part, rv := parts[i], reflect.ValueOf(v[i]).Elem()
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			fval, err := strconv.ParseFloat(part, 64)
			if err == nil {
				rv.SetFloat(fval)
			} else {
				ve = append(ve, err)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ival, err := strconv.ParseInt(part, 10, 64)
			if err == nil {
				rv.SetInt(ival)
			} else {
				ve = append(ve, err)
			}
		default:
			ve = append(ve, errors.New("unknown type in fill: " + rv.Kind().String()))
		}
	}
	return ve
}

