package server

import (
	"time"

	"net/http/pprof"

	em "github.com/BasedDevelopment/eve/pkg/middleware"
	"github.com/ezrizhu/bgprestlg/internal/config"
	"github.com/ezrizhu/bgprestlg/internal/server/routes"
	"github.com/go-chi/chi/v5"
	cm "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func Handler() *chi.Mux {
	r := chi.NewMux()

	//Middlewares
	if config.Config.API.BehindProxy {
		r.Use(cm.RealIP)
	}
	r.Use(cm.RequestID)
	r.Use(em.Logger)
	r.Use(cm.GetHead)
	r.Use(httprate.LimitByIP(100, 1*time.Minute))
	r.Use(cm.AllowContentType("application/json"))
	r.Use(cm.CleanPath)
	r.Use(cm.NoCache)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(cm.Heartbeat("/"))
	r.Use(em.Recoverer)

	r.Get("/status", routes.GetStatus)
	r.Get("/route/{prefix}", routes.GetRoute)
	r.Get("/debug/pprof", pprof.Index)
	r.Get("/debug/pprof/", pprof.Index)
	r.Get("/debug/pprof/cmdline", pprof.Cmdline)
	r.Get("/debug/pprof/profile", pprof.Profile)
	r.Get("/debug/pprof/symbol", pprof.Symbol)
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))

	return r
}
