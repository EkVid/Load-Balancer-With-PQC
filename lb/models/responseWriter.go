// this file is a wrapper to write response, can be ignored
package models

import "net/http"

type safeResponseWriter struct {
    http.ResponseWriter
    wroteHeader bool
}

func (w *safeResponseWriter) WriteHeader(status int) {
    if w.wroteHeader {
        return
    }
    w.wroteHeader = true
}

func (w *safeResponseWriter) Write(b []byte) (int, error) {
    return w.ResponseWriter.Write(b)
}
