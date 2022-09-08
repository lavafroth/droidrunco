package meta

import (
	"encoding/json"
	_ "embed"
)

//go:embed db.json
var dbContents []byte

type Meta struct {
	Package     string `json:"pkg"`
	Description string `json:"description"`
	Removal     string `json:"removal"`
}

type DB []*Meta

func (db DB) Get(pkg string) *Meta {
	for _, entry := range db {
		if entry.Package == pkg {
			return entry

		}
	}
	return nil
}

func Init() (DB, error) {
	var db DB
	if err := json.Unmarshal(dbContents, &db); err != nil {
		return nil, err
	}
	return db, nil
}
