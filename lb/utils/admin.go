// this file exposes an /admin endpoint to allow dynamic modification of backend servers
package utils

import (
	"fmt"
	"log"
	"net/http"
	"lb/models"
)

// StartAdminServer runs the admin API on :9090 to add/remove backends dynamically.
func StartAdminServer(lb *models.LoadBalancer) {
	http.HandleFunc("/admin/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		if err := lb.AddBackend(rawURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "Added backend: %s\n", rawURL)
	})

	http.HandleFunc("/admin/remove", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		lb.RemoveBackend(rawURL)
		fmt.Fprintf(w, "Removed backend: %s\n", rawURL)
	})

	http.HandleFunc("/admin/enable", func(w http.ResponseWriter, r *http.Request){
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "Missing URL parameter", http.StatusBadRequest)
			return
		}

		err := lb.EnableBackend(rawURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "Enabled backend: %s\n", rawURL)
	})

	go func() {
		fmt.Println("Admin API started on :9090")
		log.Fatal(http.ListenAndServe(":9090", nil))
	}()
}
