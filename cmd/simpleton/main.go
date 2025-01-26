package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/kluzzebass/simpleton"
)

type Settings struct {
	ListenAddrs   string
	AccessLogPath string
	ErrorLogPath  string
	Chroot        bool
	HelpFlag      bool
	ContentPath   string
}

func (s *Settings) ParseFlags() {
	flag.StringVar(&s.ListenAddrs, "l", "0.0.0.0:80,[::]:80", "Comma-separated list of addresses to listen on")
	flag.StringVar(&s.AccessLogPath, "a", "-", "Path to access log file")
	flag.StringVar(&s.ErrorLogPath, "e", "-", "Path to error log file")
	flag.BoolVar(&s.Chroot, "c", false, "Chroot to content directory")
	flag.BoolVar(&s.HelpFlag, "h", false, "Show help")
	flag.Parse()

	if s.HelpFlag || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(0)
	}

	s.ContentPath = flag.Arg(0)
}

func (s *Settings) ParseEnv() {
	listenAddrs := os.Getenv("SIMPLETON_LISTEN_ADDRS")
	if listenAddrs != "" {
		s.ListenAddrs = listenAddrs
	}

	accessLogPath := os.Getenv("SIMPLETON_ACCESS_LOG_PATH")
	if accessLogPath != "" {
		s.AccessLogPath = accessLogPath
	}

	errorLogPath := os.Getenv("SIMPLETON_ERROR_LOG_PATH")
	if errorLogPath != "" {
		s.ErrorLogPath = errorLogPath
	}

	chroot := strings.ToLower(os.Getenv("SIMPLETON_CHROOT"))
	chrootBool, err := strconv.ParseBool(chroot)
	if err == nil {
		s.Chroot = chrootBool
	}

	contentPath := os.Getenv("SIMPLETON_CONTENT_PATH")
	if contentPath != "" {
		s.ContentPath = contentPath
	}
}

func main() {
	settings := &Settings{}
	settings.ParseFlags()
	settings.ParseEnv()

	addrList := strings.Split(settings.ListenAddrs, ",")

	accessLogFile := os.Stdout
	if settings.AccessLogPath != "-" {
		err := os.MkdirAll(filepath.Dir(settings.AccessLogPath), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create access log directory: %v\n", err)
			return
		}
		accessLogFile, err = os.OpenFile(settings.AccessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open access log file: %v\n", err)
			return
		}
	}

	errorLogFile := os.Stderr
	if settings.ErrorLogPath != "-" {
		err := os.MkdirAll(filepath.Dir(settings.ErrorLogPath), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create error log directory: %v\n", err)
			return
		}
		errorLogFile, err = os.OpenFile(settings.ErrorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open error log file: %v\n", err)
			return
		}
	}

	// Chroot to content directory
	if settings.Chroot {
		if err := syscall.Chroot(settings.ContentPath); err != nil {
			fmt.Fprintf(errorLogFile, "Failed to chroot to %s: %v\n", settings.ContentPath, err)
			return
		}
		settings.ContentPath = "/"
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
			server, err := simpleton.New(addr, settings.ContentPath)
			if err != nil {
				fmt.Fprintf(errorLogFile, "Failed to create server: %v\n", err)
				return
			}
			server.
				SetAccessLog(accessLogFile).
				SetErrorLog(errorLogFile)
			if err := server.Serve(ctx); err != nil {
				fmt.Fprintf(errorLogFile, "Failed to start server on %s: %v\n", addr, err)
				return
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
	fmt.Fprintf(errorLogFile, "All servers stopped\n")
}
