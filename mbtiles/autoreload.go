package mbtiles

import (
	"log"
	"os"
	"time"
)

func (mbt *Map) SetAutoReload(autoreload bool) {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	if (mbt.ar != nil) == autoreload {
		return
	}
	if mbt.ar != nil {
		close(mbt.ar)
		mbt.ar = nil
		return
	}
	ch := make(chan bool)
	mbt.ar = ch

	go func() {
		tick := time.Tick(time.Second)
		for {
			select {
			case <-ch:
				return
			case <-tick:
				fi, err := os.Stat(mbt.Filename)
				if err != nil && fi.ModTime() != mbt.Mtime {
					mbt.mtx.Lock()
					// check if we were closed in the meantime
					if mbt.db != nil {
						var a, tmp mapsql
						var b *mapsql
						a, *b = *b, a
						mtime, err := tmp.open(mbt.Filename)
						if err == nil {
							tmp, mbt.mapsql = mbt.mapsql, tmp
							mbt.Mtime = mtime
							log.Println("database reloaded:", mtime)
						}
						tmp.close()
					}
					mbt.mtx.Unlock()
				}
			}
		}
	}()
}

