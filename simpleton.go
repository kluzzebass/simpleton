package simpleton

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Simpleton struct {
	listenAddr string
	absPath    string
	accessLog  io.Writer
	errorLog   io.Writer
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
		accessLog:  os.Stdout,
		errorLog:   os.Stderr,
	}

	return s, nil
}

func (s *Simpleton) SetAccessLog(logger io.Writer) *Simpleton {
	s.accessLog = logger
	return s
}

func (s *Simpleton) SetErrorLog(logger io.Writer) *Simpleton {
	s.errorLog = logger
	return s
}

func (s *Simpleton) log(remoteAddr, method, path, proto string, statusCode, contentLength int, startTime time.Time) {
	fmt.Fprintf(s.accessLog, "%s - - [%s] \"%s %s %s\" %d %d\n",
		remoteAddr,
		startTime.Format("02/Jan/2006:15:04:05 -0700"),
		method,
		path,
		proto,
		statusCode,
		contentLength,
	)
}

func (s *Simpleton) Serve(ctx context.Context) error {
	server := &http.Server{Addr: s.listenAddr}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		filePath := filepath.Clean(s.absPath + r.URL.Path)
		ww := &captureWriter{w: w}
		http.ServeFile(ww, r, filePath)
		s.log(r.RemoteAddr, r.Method, r.URL.Path, r.Proto, ww.statusCode, ww.contentLength, startTime)
	})

	go func() {
		fmt.Fprintf(s.errorLog, "Serving / on %s\n", s.listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(s.errorLog, "Server failed on %s: %v\n", s.listenAddr, err)
		}
	}()

	<-ctx.Done()
	fmt.Fprintf(s.errorLog, "Shutting down server on %s\n", s.listenAddr)
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
