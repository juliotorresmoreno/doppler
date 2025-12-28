package middleware

import "net/http"

func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With,Content-Type,Cache-Control")

		if req.Method == "OPTIONS" {
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			return
		}
		next.ServeHTTP(w, req)
	})
}
