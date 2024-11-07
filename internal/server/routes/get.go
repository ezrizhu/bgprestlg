package routes

import (
	"net/http"

	"github.com/ezrizhu/bgprestlg/internal/bgp"
)

func GetRoute(w http.ResponseWriter, r *http.Request) {
	bgpState := bgp.PeerState()
	//w.Write([]byte("hi"))
	w.Write([]byte(bgpState))
}
