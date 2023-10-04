package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"net/http"
	"os"
)

var machineName string

func rootHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, fmt.Sprintf("OK - EMPTY - %s", machineName))
	w.WriteHeader(http.StatusOK)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type Multiplexer struct {
	handlers map[string]func(w http.ResponseWriter, r *http.Request)
	Handler  http.HandlerFunc
}

func NewMultiplexer() *Multiplexer {
	multiplexer := &Multiplexer{
		handlers: make(map[string]func(w http.ResponseWriter, r *http.Request)),
	}
	multiplexer.init()
	return multiplexer
}

func (mux *Multiplexer) init() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// you can allow only http2 request
		//if r.ProtoMajor != 2 {
		//	log.Println("HTTP/2 request is rejected")
		//	w.WriteHeader(http.StatusInternalServerError)
		//	return
		//}

		f := mux.GetHandler(r.URL.Path)
		if f == nil {
			log.Println("unknown path", r.URL.Path)
			return
		}
		f(w, r)
	})
	mux.Handler = handler
}

func (mux *Multiplexer) GetHandler(path string) func(w http.ResponseWriter, r *http.Request) {
	f, ok := mux.handlers[path]
	if ok {
		return f
	}
	return nil
}

func (mux *Multiplexer) HandleFunc(path string, f func(w http.ResponseWriter, r *http.Request)) {
	mux.handlers[path] = f
}

func StartHTTPServer() {
	var err error
	machineName, err = os.Hostname()
	if err != nil {
		log.Fatal("Failed to get HOSTNAME environmental variable.")
	}

	mux := NewMultiplexer()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:80",
		Handler: h2c.NewHandler(mux.Handler, &http2.Server{}),
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal("Failed to set up an HTTP server - ", err)
	}
}
