
mbtilesrv
#########

Simple mbtiles file server written in Go. Also generates a
simple html map with modestmaps or leaflet in addition to 
serving the tiles from the mbtiles sqlite database.

Installation
============

Having go installed, simply build with the go tool and run it with::

    export GOPATH=~/src/gocode
    go build github.com/tajtiattila/mbtilesrv
    $GOPATH/bin/mbtilesrv map.mbtiles

Features
========

* Tile server
* Serve map html
* Detects file changes and reloads database if necessary

External dependencies
=====================

Mbtilesrv depends on gosqlite_ and freetype-go_. Install them with go get::

    go get code.google.com/p/gosqlite
    go get code.google.com/p/freetype-go

Todo
====

- Serve map (POI) data
- Search?


.. _gosqlite: http://code.google.com/p/gosqlite/
.. _freetype-go: http://code.google.com/p/freetype-go/
