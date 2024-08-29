package main

import (
	"github.com/nveeser/corepxe/server"
	"log"
	"os"
)

var srv server.IPXE

var defaultConfigDir = "/home/nicholas/pxe-files/configs/"
var defaultImageDir = "/home/nicholas/pxe-files/images/"
var defaultListenAddr = "0.0.0.0:8086"

func init() {
	srv.ConfigDir = os.Getenv("COREPXE_SERVER_CONFIG_DIR")
	if srv.ConfigDir == "" {
		srv.ConfigDir = defaultConfigDir
	}
	srv.ImageDir = os.Getenv("COREPXE_SERVER_IMAGE_DIR")
	if srv.ImageDir == "" {
		srv.ImageDir = defaultImageDir
	}
	srv.ListenAddr = os.Getenv("COREPXE_SERVER_LISTEN_ADDR")
	if srv.ListenAddr == "" {
		srv.ListenAddr = defaultListenAddr
	}
}

func main() {
	log.Fatal(srv.Run())
}
