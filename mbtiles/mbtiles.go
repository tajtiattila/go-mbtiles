package mbtiles

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

type mapsql struct {
	db       *sql.DB
	tileStmt, gridStmt, gridDataStmt *sql.Stmt
	metadata *Metadata
}

func (ms *mapsql) open(fn string) (time.Time, error) {
	fi, err := os.Stat(fn)
	var tnil time.Time
	if err != nil {
		return tnil, err
	}
	mtime := fi.ModTime()

	ok := false
	defer func() {
		if !ok {
			if ms.tileStmt != nil {
				ms.tileStmt.Close()
			}
			if ms.gridStmt != nil {
				ms.gridStmt.Close()
			}
			if ms.gridDataStmt != nil {
				ms.gridDataStmt.Close()
			}
			if ms.db != nil {
				ms.db.Close()
			}
		}
	}()

	ms.db, err = sql.Open("sqlite3", fn)
	if err != nil {
		return tnil, err
	}
	ms.metadata, err = mbtMetadata(ms.db)
	if err != nil {
		return tnil, err
	}
	ms.tileStmt, err = ms.db.Prepare(`select tile_data from tiles
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return tnil, err
	}
	ms.gridStmt, err = ms.db.Prepare(`select grid from grids
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return tnil, err
	}
	ms.gridDataStmt, err = ms.db.Prepare(`select key_name,key_json from grid_data
where zoom_level = ?1 and tile_column = ?2 and tile_row = ?3`)
	if err != nil {
		return tnil, err
	}
	ok = true
	return mtime, err
}

func (ms *mapsql) close() error {
	ms.tileStmt.Close()
	ms.gridStmt.Close()
	ms.gridDataStmt.Close()
	err := ms.db.Close()
	ms.db = nil
	ms.tileStmt = nil
	ms.gridStmt = nil
	ms.gridDataStmt = nil
	return err
}

type Map struct {
	mapsql
	Filename string
	Mtime    time.Time
	mtx      sync.Mutex
	ar     chan<- bool
}

func Open(dbname string) (*Map, error) {
	mbt := &Map{Filename: dbname}
	var err error
	mbt.Mtime, err = mbt.open(mbt.Filename)
	if err != nil {
		return nil, err
	}
	return mbt, err
}

func (mbt *Map) Close() error {
	mbt.mtx.Lock()
	defer mbt.mtx.Unlock()
	if mbt.ar != nil {
		close(mbt.ar)
		mbt.ar = nil
	}
	return mbt.mapsql.close()
}

func (mbt *Map) GetTile(z, x, y int) ([]byte, error) {
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

func (mbt *Map) GetGridData(z, x, y int, callback string) ([]byte, error) {
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

func (mbt *Map) Metadata() *Metadata {
	return mbt.metadata
}

