package meta

import (
	"encoding/json"
	_ "embed"
)

//go:embed db.json
var dbContents []byte

type Meta struct {
	Id     string `json:"id"`
	Description string `json:"description"`
	Removal     string `json:"removal"`
	List string `json:"list"`
}

type DB []*Meta

func (db DB) Get(id string) *Meta {
	for _, entry := range db {
		if entry.Id == id {
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
