package routes

import "net/http"

func GetRoute(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi"))
}
