package server

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type IPXE struct {
	BaseUrl    string
	DataDir    string
	ListenAddr string
}

func (c *IPXE) Run() error {
	handler, err := c.buildHandler()
	if err != nil {
		return err
	}
	// Start the iPXE Boot Server.
	fmt.Println("Starting CoreOS iPXE Server...")
	fmt.Printf("Listening on %s\n", c.ListenAddr)
	if c.BaseUrl != "" {
		fmt.Printf("Base URL %s\n", c.BaseUrl)
	}
	fmt.Printf("Data directory: %s\n", c.DataDir)

	httpSrv := http.Server{
		Addr:    c.ListenAddr,
		Handler: handler,
	}
	return httpSrv.ListenAndServe()
}

func (c *IPXE) buildHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	p := "/images/"
	ih, err := NewImageHandler(filepath.Join(c.DataDir, p))
	if err != nil {
		return nil, err
	}
	mux.Handle("/images/{ostype}/{filetype}", http.StripPrefix(p, ih))

	p = "/configs/"
	handler := http.FileServer(http.Dir(filepath.Join(c.DataDir, p)))
	mux.Handle(p, http.StripPrefix(p, handler))
	return withLogging(mux), nil
}

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w}
		h.ServeHTTP(ww, r)
		duration := time.Since(start)
		log.Printf("%s %s%s %d %s", r.Method, r.URL.Path, r.URL.Query().Encode(), ww.code, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (l *responseWriter) Write(b []byte) (int, error) {
	if l.code == 0 {
		l.code = http.StatusOK
	}
	return l.ResponseWriter.Write(b)
}

func (l *responseWriter) WriteHeader(statusCode int) {
	if l.code == 0 {
		l.code = statusCode
	}
	l.ResponseWriter.WriteHeader(statusCode)
}
