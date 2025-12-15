package main

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"

	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/bridge"
	"github.com/lavafroth/droidrunco/web"

	"github.com/gorilla/websocket"
)

func main() {
	bridge.Init()
	defer bridge.Close()

	web.WsLoopHandleFunc("/list", func(conn *websocket.Conn) error {
		for {
			gotNewPackages, err := bridge.Refresh()
			if err != nil {
				return err
			}

			if gotNewPackages {
				if err := conn.WriteJSON(slices.Collect(maps.Values(bridge.Cache))); err != nil {
					return fmt.Errorf("Failed writing fresh package list to websocket connection: %q", err)
				}
			}
		}
	})
	web.WsLoopHandleFunc("/patch", func(conn *websocket.Conn) error {
		for {
			App := app.App{}
			if err := conn.ReadJSON(&App); err != nil {
				return fmt.Errorf("Failed to read patch query websocket connection: %q", err)
			}
			if err := conn.WriteJSON(map[string]string{"status": bridge.Toggle(bridge.Cache[App.Id])}); err != nil {
				return fmt.Errorf("Failed writing current state of app to websocket connection: %q", err)
			}
		}
	})
	log.Print("Visit http://localhost:8080 to access the dashboard")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
