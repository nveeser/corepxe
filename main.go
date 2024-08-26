package main

import (
	"github.com/nveeser/corepxe/server"
	"log"
	"os"
)

var srv server.IPXE

var defaultDataDir = "/home/nicholas/pxe-files"
var defaultListenAddr = "0.0.0.0:8086"

func init() {
	srv.BaseUrl = os.Getenv("COREPXE_SERVER_BASE_URL")
	srv.DataDir = os.Getenv("COREPXE_SERVER_DATA_DIR")
	if srv.DataDir == "" {
		srv.DataDir = defaultDataDir
	}
	srv.ListenAddr = os.Getenv("COREPXE_SERVER_LISTEN_ADDR")
	if srv.ListenAddr == "" {
		srv.ListenAddr = defaultListenAddr
	}
}

func main() {
	log.Fatal(srv.Run())
}
