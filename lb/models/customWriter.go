// this file is a wrapper to write response to capture status code and body
package models

import (
    "bytes"      
    "net/http"
)


type customResponseWriter struct {
    http.ResponseWriter
    header http.Header
    status int
    body   *bytes.Buffer
}

// Override methods in http.responseWriter
func (w *customResponseWriter) Header() http.Header {
    return w.header  
}

func (w *customResponseWriter) Write(b []byte) (int, error) {
    return w.body.Write(b)  
}

func (w *customResponseWriter) WriteHeader(statusCode int) {
    w.status = statusCode  
}