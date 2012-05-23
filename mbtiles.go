package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"os"
	"sync"
	"time"
)

var ErrTileNotFound = errors.New("tile does not exist")

type MBTiles struct {
	Filename string
	Mtime    time.Time
	mtx      sync.Mutex
	conn     *sqlite.Conn
	done     chan<- bool
}

func OpenMBTiles(dbname string) (*MBTiles, error) {
	mbt := &MBTiles{Filename: dbname}
	fi, err := os.Stat(mbt.Filename)
	if err != nil {
		return nil, err
	}
	mbt.Mtime = fi.ModTime()

	mbt.conn, err = sqlite.Open(mbt.Filename)
	if err != nil {
		return nil, err
	}
	return mbt, nil
}

func (mbt *MBTiles) Close() error {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	if mbt.done != nil {
		mbt.done <- true
	}
	err := mbt.conn.Close()
	mbt.conn = nil
	return err
}

func (mbt *MBTiles) AutoReload() {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	if mbt.done != nil {
		return
	}
	ch := make(chan bool)
	mbt.done = ch

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
					if mbt.conn != nil {
						nconn, err := sqlite.Open(mbt.Filename)
						if err == nil {
							mbt.conn.Close()
							mbt.Mtime = fi.ModTime()
							mbt.conn = nconn
						}
					}
					mbt.mtx.Unlock()
				}
			}
		}
	}()
}

func (mbt *MBTiles) GetTile(z, x, y int) ([]byte, error) {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	stmt, err := mbt.conn.Prepare(`select tile_data from tiles
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err == nil {
		if err = stmt.Exec(z, x, y); err == nil {
			if !stmt.Next() {
				return nil, ErrTileNotFound
			}
			var blob []byte
			if err = stmt.Scan(&blob); err == nil {
				return blob, nil
			}
		}
	}
	return nil, err
}

func (mbt *MBTiles) Metadata() (*Metadata, error) {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	return MbtMetadata(mbt.conn)
}
