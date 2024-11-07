package bgp

import (
	"context"
	"fmt"

	"github.com/ezrizhu/bgprestlg/internal/config"
	api "github.com/osrg/gobgp/v3/api"
	bgpLog "github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ctx = context.Background()
var s *server.BgpServer

func SrvInit() {
	s = server.NewBgpServer(server.LoggerOption(&myLogger{logger: &log.Logger}))
	go s.Serve()

	if err := s.StartBgp(ctx, &api.StartBgpRequest{
		Global: &api.Global{
			Asn:             uint32(config.Config.BGP.ASN),
			RouterId:        config.Config.BGP.RouterID,
			ListenPort:      int32(config.Config.BGP.Port),
			ListenAddresses: []string{config.Config.BGP.Address},
		},
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to start BGP server")
	}

	if err := s.WatchEvent(ctx, &api.WatchEventRequest{Peer: &api.WatchEventRequest_Peer{}}, func(r *api.WatchEventResponse) {
		if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_STATE {
			log.Debug().
				Str("src", "gobgp.peer").
				Msg(p.String())
		}
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to install watchEvent hook")
	}
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: config.Config.Peer.Address,
			PeerAsn:         uint32(config.Config.Peer.Port),
		},
	}

	if err := s.AddPeer(ctx, &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		log.Error().Err(err).Msg("AddPeer")
		return
	}
}

func SrvStop(ctx context.Context) error {
	if err := s.ShutdownPeer(ctx, &api.ShutdownPeerRequest{
		Address: config.Config.Peer.Address,
	}); err != nil {
		log.Error().Err(err).Msg("ShutdownPeer")
	}
	return s.StopBgp(ctx, &api.StopBgpRequest{})
}

type myLogger struct {
	logger *zerolog.Logger
}

func (l *myLogger) log(level zerolog.Level, msg string, fields bgpLog.Fields) {
	event := l.logger.WithLevel(level).Str("src", "gobgp.server")
	for key, value := range fields {
		event = event.Str(key, fmt.Sprint(value))
	}
	event.Msg(msg)
}

func (l *myLogger) Panic(msg string, fields bgpLog.Fields) {
	l.log(zerolog.PanicLevel, msg, fields)
}

func (l *myLogger) Fatal(msg string, fields bgpLog.Fields) {
	l.log(zerolog.FatalLevel, msg, fields)
}

func (l *myLogger) Error(msg string, fields bgpLog.Fields) {
	l.log(zerolog.ErrorLevel, msg, fields)
}

func (l *myLogger) Warn(msg string, fields bgpLog.Fields) {
	l.log(zerolog.WarnLevel, msg, fields)
}

func (l *myLogger) Info(msg string, fields bgpLog.Fields) {
	l.log(zerolog.InfoLevel, msg, fields)
}

func (l *myLogger) Debug(msg string, fields bgpLog.Fields) {
	l.log(zerolog.DebugLevel, msg, fields)
}

func (l *myLogger) SetLevel(level bgpLog.LogLevel) {
	l.logger.Level(zerolog.Level(level))
}

func (l *myLogger) GetLevel() bgpLog.LogLevel {
	return bgpLog.LogLevel(l.logger.GetLevel())
}
