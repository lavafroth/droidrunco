package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/bridge"
	"github.com/lavafroth/droidrunco/web"

	"github.com/gorilla/websocket"
)

func main() {
	bridge.Init()
	defer bridge.Close()
	log.Print("Indexing packages")

	// This first refresh is the most time consuming
	// as it has to index all the apps on the device
	bridge.Refresh()
	web.WsLoopHandleFunc("/list", func(conn *websocket.Conn) error {
		firstTimer := true
		for {
			// These subsequent calls are cheap, both in terms
			// of time as well as processing because we prune
			// all the packages previously seen.
			bridge.Refresh()
			if bridge.Updated || firstTimer {
				if err := conn.WriteJSON(bridge.Cache); err != nil {
					return fmt.Errorf("Failed writing fresh package list to websocket connection: %q", err)
				}
				bridge.Updated = false
			}
			firstTimer = false
		}
	})
	web.WsLoopHandleFunc("/patch", func(conn *websocket.Conn) error {
		for {
			App := app.App{}
			if err := conn.ReadJSON(&App); err != nil {
				return fmt.Errorf("Failed to read patch query websocket connection: %q", err)
			}
			if err := conn.WriteJSON(map[string]string{"status": bridge.Toggle(bridge.Cache.Get(App.Id))}); err != nil {
				return fmt.Errorf("Failed writing current state of app to websocket connection: %q", err)
			}
			bridge.Updated = true
		}
	})
	log.Print("Visit http://localhost:8080 to access the dashboard")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
