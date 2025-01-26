package simpleton

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Simpleton struct {
	listenAddr string
	absPath    string
	accessLog  *log.Logger
	errorLog   *log.Logger
	logMutex   sync.Mutex
}

func New(listenAddr string, contentDir string) (*Simpleton, error) {
	// resolve the absolute path of the content directory
	fmt.Println("contentDir: ", contentDir)
	absPath, err := filepath.Abs(contentDir)
	fmt.Println("absPath: ", absPath)
	if err != nil {
		return nil, err
	}

	// check that we can serve from the content directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		fmt.Println("os.Stat: ", err)
		return nil, err
	}

	// check that the content directory is a directory
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	s := &Simpleton{
		listenAddr: listenAddr,
		absPath:    absPath,
		accessLog:  log.Default(),
		errorLog:   log.Default(),
	}

	return s, nil
}

func (s *Simpleton) SetAccessLog(logger *log.Logger) *Simpleton {
	s.accessLog = logger
	return s
}

func (s *Simpleton) SetErrorLog(logger *log.Logger) *Simpleton {
	s.errorLog = logger
	return s
}

func (s *Simpleton) logCommonFormat(remoteAddr, method, path, proto string, statusCode, contentLength int, startTime time.Time) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	s.accessLog.Printf("%s - - [%s] \"%s %s %s\" %d %d",
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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		filePath := filepath.Clean(s.absPath + r.URL.Path)
		ww := &captureWriter{w: w}
		http.ServeFile(ww, r, filePath)
		s.logCommonFormat(r.RemoteAddr, r.Method, r.URL.Path, r.Proto, ww.statusCode, ww.contentLength, startTime)
	})

	server := &http.Server{Addr: s.listenAddr}
	go func() {
		s.errorLog.Printf("Serving / on %s", s.listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.errorLog.Printf("Server failed on %s: %v", s.listenAddr, err)
		}
	}()

	<-ctx.Done()
	s.errorLog.Println("Shutting down server...")
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
