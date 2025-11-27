type safeResponseWriter struct {
    http.ResponseWriter
    wroteHeader bool
}

func (w *safeResponseWriter) WriteHeader(status int) {
    if w.wroteHeader {
        return
    }
    w.wroteHeader = true
    w.ResponseWriter.WriteHeader(status)
}

func (w *safeResponseWriter) Write(b []byte) (int, error) {
    if !w.wroteHeader {
        w.WriteHeader(http.StatusOK)
    }
    return w.ResponseWriter.Write(b)
}
