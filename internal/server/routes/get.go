package routes

import (
	"net/http"

	"github.com/ezrizhu/bgprestlg/internal/bgp"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func GetStatus(w http.ResponseWriter, r *http.Request) {
	bgpState := bgp.PeerState()
	w.Write([]byte(bgpState))
}

func GetRoute(w http.ResponseWriter, r *http.Request) {
	prefix := chi.URLParam(r, "prefix")
	log.Debug().Msg("getting prefix" + prefix)
	bgpRoute := bgp.Route(prefix)
	w.Write([]byte(bgpRoute))
}
