package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"silk-controller/handlers"
	"silk-controller/store"

	"code.cloudfoundry.org/lager"
)

func main() {
	logger := lager.NewLogger("container-networking.silk-controller")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	var listenPort int
	flag.IntVar(&listenPort, "listenPort", 5000, "port to listen on")
	flag.Parse()

	dataStore := store.NewDatastore()

	leaseHandlers := handlers.Leases{
		Logger: logger,
		Store:  dataStore,
	}

	router, err := leaseHandlers.BuildRouter()
	if err != nil {
		logger.Fatal("rata-new-router", err)
	}

	(&http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: router,
	}).ListenAndServe()
}
