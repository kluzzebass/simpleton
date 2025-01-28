package simpleton

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Simpleton struct {
	listenAddr string
	absPath    string
	accessLog  *slog.Logger
	serverLog  *slog.Logger
	ncsaLog    *NCSAHandler
}

func New(listenAddr string, contentDir string) (*Simpleton, error) {
	// resolve the absolute path of the content directory
	absPath, err := filepath.Abs(contentDir)
	if err != nil {
		return nil, err
	}

	// check that we can serve from the content directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	// check that the content directory is a directory
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory\n", absPath)
	}

	s := &Simpleton{
		listenAddr: listenAddr,
		absPath:    absPath,
		accessLog:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
		serverLog:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	return s, nil
}

func (s *Simpleton) SetAccessLog(file *os.File) *Simpleton {
	s.accessLog = slog.New(NewNCSAHandler(file))
	return s
}

func (s *Simpleton) SetServerLog(file *os.File) *Simpleton {
	s.serverLog = slog.New(slog.NewTextHandler(file, nil))
	return s
}

func (s *Simpleton) log(remoteAddr, method, path, proto string, statusCode, contentLength int) {
	s.accessLog.Info(
		"access",
		"host", remoteAddr,
		"user", "-",
		"method", method,
		"path", path,
		"protocol", proto,
		"status", statusCode,
		"size", contentLength,
	)
}

func (s *Simpleton) Serve(ctx context.Context) error {
	server := &http.Server{Addr: s.listenAddr}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Clean(s.absPath + r.URL.Path)
		ww := &captureWriter{w: w}
		http.ServeFile(ww, r, filePath)
		s.log(r.RemoteAddr, r.Method, r.URL.Path, r.Proto, ww.statusCode, ww.contentLength)
	})

	go func() {
		s.serverLog.Info("serving", "listenAddr", s.listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.serverLog.Error("server failed", "listenAddr", s.listenAddr, "error", err)
		}
	}()

	<-ctx.Done()
	s.serverLog.Info("shutting down", "listenAddr", s.listenAddr)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// captureWriter is a http.ResponseWriter that captures the status code and content length of the response.
type captureWriter struct {
	w             http.ResponseWriter
	statusCode    int
	contentLength int
}

func (r *captureWriter) Header() http.Header {
	return r.w.Header()
}

func (r *captureWriter) Write(b []byte) (int, error) {
	r.contentLength += len(b)
	return r.w.Write(b)
}

func (r *captureWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.w.WriteHeader(statusCode)
}
