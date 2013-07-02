package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"sync"
	"time"
)

var ErrTileNotFound = errors.New("tile does not exist")

type MBTiles struct {
	Filename string
	Mtime    time.Time
	mtx      sync.Mutex
	db       *sql.DB
	tileStmt, gridStmt, gridDataStmt *sql.Stmt
	done     chan<- bool
}

func OpenMBTiles(dbname string) (*MBTiles, error) {
	mbt := &MBTiles{Filename: dbname}
	fi, err := os.Stat(mbt.Filename)
	if err != nil {
		return nil, err
	}
	mbt.Mtime = fi.ModTime()

	ok := false
	defer func() {
		if !ok {
			if mbt.tileStmt != nil {
				mbt.tileStmt.Close()
			}
			if mbt.gridStmt != nil {
				mbt.gridStmt.Close()
			}
			if mbt.gridDataStmt != nil {
				mbt.gridDataStmt.Close()
			}
			if mbt.db != nil {
				mbt.db.Close()
			}
		}
	}()

	mbt.db, err = sql.Open("sqlite3", mbt.Filename)
	if err != nil {
		return nil, err
	}
	mbt.tileStmt, err = mbt.db.Prepare(`select tile_data from tiles
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return nil, err
	}
	mbt.gridStmt, err = mbt.db.Prepare(`select grid from grids
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return nil, err
	}
	mbt.gridDataStmt, err = mbt.db.Prepare(`select key_name,key_json from grid_data
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return nil, err
	}

	ok = true
	return mbt, nil
}

func (mbt *MBTiles) Close() error {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	if mbt.done != nil {
		mbt.done <- true
	}
	mbt.tileStmt.Close()
	mbt.gridStmt.Close()
	mbt.gridDataStmt.Close()
	err := mbt.db.Close()
	mbt.db = nil
	mbt.tileStmt = nil
	mbt.gridStmt = nil
	mbt.gridDataStmt = nil
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
					if mbt.db != nil {
						nconn, err := sql.Open("sqlite3", mbt.Filename)
						if err == nil {
							mbt.db.Close()
							mbt.Mtime = fi.ModTime()
							mbt.db = nconn
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
	rows, err := mbt.tileStmt.Query(z, x, y)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrTileNotFound
	}
	var blob []byte
	if err = rows.Scan(&blob); err == nil {
		return blob, nil
	}
	return nil, err
}

func (mbt *MBTiles) GetGridData(z, x, y int, callback string) ([]byte, error) {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	rows, err := mbt.gridStmt.Query(z, x, y)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrTileNotFound
	}
	var blob []byte
	if err = rows.Scan(&blob); err != nil {
		return nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	gd := make(map[string]*json.RawMessage)
	if err = json.NewDecoder(zr).Decode(&gd); err != nil {
		return nil, err
	}
	rows, err = mbt.gridDataStmt.Query(z, x, y)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var data bytes.Buffer
	sep := ""
	data.WriteString("{")
	for rows.Next() {
		var key_name, key_json string
		if err = rows.Scan(&key_name, &key_json); err != nil {
			break
		}
		data.WriteString(sep + `"` + key_name + `":` + key_json)
		sep = ","
	}
	data.WriteString("}")
	datamsg := json.RawMessage(data.Bytes())
	gd["data"] = &datamsg

	if callback != "" {
		var final bytes.Buffer
		final.WriteString(callback + "(")
		err = json.NewEncoder(&final).Encode(gd)
		final.WriteString(");")
		return final.Bytes(), err
	}

	return json.Marshal(gd)
}

func (mbt *MBTiles) Metadata() (*Metadata, error) {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	return MbtMetadata(mbt.db)
}
