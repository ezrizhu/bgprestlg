package bgp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ezrizhu/bgprestlg/internal/config"
	api "github.com/osrg/gobgp/v3/api"
	bgpLog "github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ctx = context.Background()
var s *server.BgpServer
var peer *api.Peer

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
	peer = &api.Peer{
		EbgpMultihop: &api.EbgpMultihop{
			Enabled:     true,
			MultihopTtl: uint32(255),
		},
		Conf: &api.PeerConf{
			NeighborAddress: config.Config.Peer.Address,
			PeerAsn:         uint32(config.Config.Peer.ASN),
		},
	}

	if err := s.AddPeer(ctx, &api.AddPeerRequest{
		Peer: peer,
	}); err != nil {
		log.Error().Err(err).Msg("AddPeer")
		return
	}
}

func SessionStateToString(state api.PeerState_SessionState) string {
	switch state {
	case api.PeerState_UNKNOWN:
		return "UNKNOWN"
	case api.PeerState_IDLE:
		return "IDLE"
	case api.PeerState_CONNECT:
		return "CONNECT"
	case api.PeerState_ACTIVE:
		return "ACTIVE"
	case api.PeerState_OPENSENT:
		return "OPENSENT"
	case api.PeerState_OPENCONFIRM:
		return "OPENCONFIRM"
	case api.PeerState_ESTABLISHED:
		return "ESTABLISHED"
	default:
		return "INVALID"
	}
}

func msgToString(m *api.Message) string {
	return fmt.Sprintf(
		"Notification: %d\n"+
			"Update: %d\n"+
			"Open: %d\n"+
			"Keepalive: %d\n"+
			"Refresh: %d\n"+
			"Discarded: %d\n"+
			"Total: %d\n"+
			"WithdrawUpdate: %d\n"+
			"WithdrawPrefix: %d",
		m.Notification,
		m.Update,
		m.Open,
		m.Keepalive,
		m.Refresh,
		m.Discarded,
		m.Total,
		m.WithdrawUpdate,
		m.WithdrawPrefix,
	)
}

func PeerState() string {
	stateStr := "State: "
	flopsStr := "Flops: "
	recvStr := "Recv: "
	sentStr := "sent: "
	if peer == nil {
		return "peer not initialized"
	}
	if _, err := s.UpdatePeer(ctx, &api.UpdatePeerRequest{
		Peer: peer,
	}); err != nil {
		return "update peer failed"
	}
	state := peer.GetState()
	if state == nil {
		return "peer state doesnt exist"
	}
	stateStr += SessionStateToString(state.SessionState)
	flopsStr += strconv.Itoa(int(state.Flops))
	if state.Messages != nil {
		if state.Messages.Received != nil {
			recvStr += msgToString(state.Messages.Received)
		}
		if state.Messages.Sent != nil {
			sentStr += msgToString(state.Messages.Sent)
		}
	}
	return stateStr + "\n" + flopsStr + "\n" + recvStr + "\n" + sentStr
}

func Route(prefix string, prefixLen string) string {
	v6Family := &api.Family{
		Afi:  api.Family_AFI_IP6,
		Safi: api.Family_SAFI_UNICAST,
	}
	v4Family := &api.Family{
		Afi:  api.Family_AFI_IP,
		Safi: api.Family_SAFI_UNICAST,
	}

	log.Info().
		Str("prefix", prefix).
		Str("len", prefixLen).
		Msg("Looking up")

	resp := ""
	if strings.Contains(prefix, ":") {
		// ipv6
		s.ListPath(context.Background(), &api.ListPathRequest{
			Family: v6Family,
			Prefixes: []*api.TableLookupPrefix{
				{
					Prefix: prefix + prefixLen,
					Type:   api.TableLookupPrefix_EXACT,
				},
			},
		}, func(p *api.Destination) {
			resp += p.Prefix
		})
	} else {
		s.ListPath(context.Background(), &api.ListPathRequest{
			Family: v4Family,
			Prefixes: []*api.TableLookupPrefix{
				{
					Prefix: prefix + prefixLen,
					Type:   api.TableLookupPrefix_EXACT,
				},
			},
		}, func(p *api.Destination) {
			resp += p.Prefix
		})
	}
	return resp
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
