package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/go-plugins-helpers/volume"
)

const (
	driverID      = "gcstorage"
	driverTCPPort = "localhost:8080"
)

var serviceKeyPath = flag.String("gcp-key-json", "", "Google Cloud Platform Service Account Key as JSON")

func main() {
	// define CLI & get args
	var Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(*serviceKeyPath) == 0 {
		Usage()
		os.Exit(1)
	}

	// define volume driver
	defaultPath := filepath.Join(volume.DefaultDockerRootDirectory, driverID)
	gcpServiceKeyAbsPath, err := filepath.Abs(*serviceKeyPath)
	if err != nil {
		log.Fatal(err)
	}
	volDriver, err := newGcpVolDriver(defaultPath, gcpServiceKeyAbsPath)
	if err != nil {
		log.Fatal(err)
	}

	// create volume handler
	volHandler := volume.NewHandler(volDriver)

	// start HTTP server
	if runtime.GOOS == "linux" {
		log.Printf("Listening on unix socket /run/docker/plugins/%s.sock...\n", driverID)
		log.Println(volHandler.ServeUnix("root", driverID))
	}
	if runtime.GOOS == "darwin" { // MacOS
		log.Fatal("unix socket creation is only supported on linux and freebsd")
		//TODO
		// log.Printf("TCP server listening on port %s...\n", driverTCPPort)
		// log.Println(volHandler.ServeTCP(driverID, driverTCPPort))
	}
}
