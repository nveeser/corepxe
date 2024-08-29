package server

import (
	"fmt"
	"github.com/nveeser/corepxe/coreos"
	"github.com/nveeser/corepxe/ignition"
	"github.com/nveeser/corepxe/mirror"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type IPXE struct {
	ConfigDir  string
	ImageDir   string
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
	fmt.Printf("Configs: %s\n", c.ConfigDir)
	fmt.Printf("Images: %s\n", c.ImageDir)

	httpSrv := http.Server{
		Addr:    c.ListenAddr,
		Handler: handler,
	}
	return httpSrv.ListenAndServe()
}

func (c *IPXE) buildHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	ih := &coreos.ImageHandler{
		ImageMirror: &mirror.ImageMirror{
			RootDir: filepath.Join(c.ImageDir, "/coreos/"),
		},
		Streams: &coreos.StreamCache{
			LocalDir: filepath.Join(c.ImageDir, "/coreos/"),
		},
	}
	mux.Handle("GET /images/coreos/{filetype}", ih)

	ignHandler := &ignition.Handler{
		ConfigRoot: c.ConfigDir,
	}
	mux.Handle("GET /configs/{osname}/{name}", ignHandler)

	pxeHandler, err := NewIPXEHandler(c.ConfigDir)
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /configs/ipxe/{name}", pxeHandler)

	mux.Handle("/configs/", http.StripPrefix("/configs/", http.FileServer(http.Dir(c.ConfigDir))))
	return withLogging(mux), nil
}

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusRespWriter{ResponseWriter: w}
		h.ServeHTTP(ww, r)
		duration := time.Since(start)
		log.Printf("%s %s%s %d %s", r.Method, r.URL.Path, r.URL.Query().Encode(), ww.code, duration)
	})
}

type statusRespWriter struct {
	http.ResponseWriter
	code int
}

func (l *statusRespWriter) Write(b []byte) (int, error) {
	if l.code == 0 {
		l.code = http.StatusOK
	}
	return l.ResponseWriter.Write(b)
}

func (l *statusRespWriter) WriteHeader(statusCode int) {
	if l.code == 0 {
		l.code = statusCode
	}
	l.ResponseWriter.WriteHeader(statusCode)
}
