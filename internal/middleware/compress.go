package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "failed to decompress", http.StatusBadRequest)
			return
		}
		defer gr.Close()

		r.Body = gr
		next.ServeHTTP(w, r)
	})
}

func CompressMiddleware(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			grw := newGzipResponseWriter(w)
			defer func() {
				if err := grw.Close(); err != nil {
					logger.Errorf("failed to close gzip writer: %v", err)
				}
			}()

			next.ServeHTTP(grw, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	w.Header().Set("Content-Encoding", "gzip")
	return &gzipResponseWriter{
		ResponseWriter: w,
		writer:         gzip.NewWriter(w),
	}
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return err
		}
		return w.writer.Close()
	}
	return nil
}
