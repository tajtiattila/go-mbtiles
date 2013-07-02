
mbtilesrv
#########

Simple mbtiles file server written in Go. Also generates a
simple html map with modestmaps or leaflet in addition to
serving the tiles from the mbtiles sqlite database.

Installation
============

Having go installed, simply build with the go tool and run it with::

    export GOPATH=~/go
    go build github.com/tajtiattila/mbtilesrv
    $GOPATH/bin/mbtilesrv map.mbtiles

Features
========

* Tile server
* Serve map html
* Detects file changes and reloads database if necessary
* UTFGrid and TileJSON support

External dependencies
=====================

Mbtilesrv depends on go-sqlite3_ and freetype-go_. Install them with go get::

    go get github.com/mattn/go-sqlite3
    go get code.google.com/p/freetype-go

Todo
====

- Serve map (POI) data
- Search?


.. _go-sqlite3: http://github.com/mattn/go-sqlite3
.. _freetype-go: http://code.google.com/p/freetype-go/
