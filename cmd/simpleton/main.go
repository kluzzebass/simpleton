package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/kluzzebass/simpleton"
)

func main() {
	listenAddrs := flag.String("l", "0.0.0.0:80,[::]:80", "Comma-separated list of addresses to listen on")
	accessLogPath := flag.String("a", "-", "Path to access log file")
	errorLogPath := flag.String("e", "-", "Path to error log file")
	chroot := flag.Bool("c", false, "Chroot to content directory")
	helpFlag := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *helpFlag || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(0)
	}

	contentPath := flag.Arg(0)
	addrList := strings.Split(*listenAddrs, ",")

	accessLogFile := os.Stdout
	if *accessLogPath != "-" {
		var err error
		accessLogFile, err = os.OpenFile(*accessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open access log file: %v", err)
		}
	}
	accessLog := log.New(accessLogFile, "", log.LstdFlags)

	errorLogFile := os.Stderr
	if *errorLogPath != "-" {
		var err error
		errorLogFile, err = os.OpenFile(*errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open error log file: %v", err)
		}
	}
	errorLog := log.New(errorLogFile, "", log.LstdFlags)

	// Chroot to content directory
	if *chroot {
		if err := syscall.Chroot(contentPath); err != nil {
			log.Fatalf("Failed to chroot to %s: %v", contentPath, err)
		}
		contentPath = "/"
	}

	// Set up a context that can be used to cancel the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn a server for each address
	wg := sync.WaitGroup{}
	for _, addr := range addrList {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			server, err := simpleton.New(addr, contentPath)
			if err != nil {
				log.Fatalf("Failed to create server: %v", err)
			}
			server.
				SetAccessLog(accessLog).
				SetErrorLog(errorLog)
			if err := server.Serve(ctx); err != nil {
				log.Fatalf("Failed to start server on %s: %v", addr, err)
			}
		}(addr)
	}

	// Listen for signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()

	// Wait for servers to stop
	wg.Wait()
	log.Println("All servers stopped")
}
