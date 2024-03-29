package web

import (
	"embed"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

//go:embed templates/index.html
var index []byte

//go:embed assets/*
var assets embed.FS

func init() {
	http.Handle("/public/", http.StripPrefix(strings.TrimRight("/public/", "/"), http.FileServer(http.FS(assets))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	})
}

func WsLoopHandleFunc(path string, Fn func(conn *websocket.Conn) error) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Printf("Failed upgrading to websocket: %q", err)
			}
			return
		}
		defer conn.Close()
		if err := Fn(conn); err != nil {
			log.Print(err)
			return
		}
	})
}
