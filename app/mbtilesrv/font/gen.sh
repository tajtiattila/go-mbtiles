#!/bin/bash

# run this script to update font.go if the luxi sans font file is updated

$GOPATH/bin/go-bindata -i=luxisr.ttf -f=luxiSansFontData -o=../font.go
