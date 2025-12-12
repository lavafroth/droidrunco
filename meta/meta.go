package meta

import (
	_ "embed"
	"encoding/json"
)

//go:embed db.json
var dbContents []byte

type Meta struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Removal     string `json:"removal"`
	List        string `json:"list"`
}

type DB map[string]*Meta

func (db DB) Get(id string) *Meta {
	return db[id]
}

func Init() (DB, error) {
	var db DB
	if err := json.Unmarshal(dbContents, &db); err != nil {
		return nil, err
	}
	return db, nil
}
