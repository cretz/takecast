package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/cretz/takecast/pkg/cert"
	"github.com/cretz/takecast/pkg/receiver"
	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
)

type Config struct {
	// If empty, uses receiver.NopLog
	Log receiver.Log
	// Used and must be present if IntermediateCACerts is nil/empty
	RootCACert *cert.KeyPair
	// If nil/empty, one is generated. The peer and auth certs are created from
	// the first one if present.
	IntermediateCACerts []*cert.KeyPair
	// If empty, generated from first/created intermediate
	PeerCert *cert.KeyPair
	// If empty, generated from first/created intermediate
	AuthCert *cert.KeyPair
	// If empty, it is "tcp"
	TLSListenNetwork string
	// If empty, it is ":0"
	TLSListenAddr string
	// If empty, it is created with other data. If present, it will not be closed
	// on close.
	TLSListenerOverride net.Listener
	// If empty, is "TakeCast"
	BroadcastInstanceName string
	// If empty, is "TakeCast"
	BroadcastFriendlyName string
	// If any value here is empty, it is considered a delete
	BroadcastTextOverrides map[string]string
	// If empty, uses all
	BroadcastIfaces []net.Interface
	// If empty, it is created with other data above. If present, it will not be
	// shutdown on close.
	BroadcastServerOverride *zeroconf.Server
	// If empty, uses generated uuid v4 with dashes removed
	ID string
	// If empty, uses receiver.NewConn
	NewConn func(receiver.ConnConfig) (receiver.Conn, error)
	// If empty, uses default receiver. Result from here will not be closed on
	// Close.
	ReceiverForConn func(receiver.Conn) (receiver.Receiver, error)
}

// Do not re-assign any fields here
type Server struct {
	Config
	TLSListener     net.Listener
	BroadcastServer *zeroconf.Server
	// Nil if Config.ReceiverForConn is present
	Receiver receiver.Receiver

	ctx    context.Context
	cancel context.CancelFunc
}

func Listen(config Config) (*Server, error) {
	s := &Server{Config: config}
	if s.Log == nil {
		s.Log = receiver.NopLog()
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	success := false
	defer func() {
		if !success {
			s.Close()
		}
	}()
	// Create the intermediate cert if necessary
	if len(s.IntermediateCACerts) == 0 {
		if s.RootCACert == nil {
			return nil, fmt.Errorf("RootCACert is required if IntermediateCACerts is empty")
		}
		s.Log.Debugf("Generating intermediate CA certs")
		interCert, err := cert.GenerateIntermediateCAKeyPair(s.RootCACert, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed generating intermediate cert: %w", err)
		}
		s.IntermediateCACerts = []*cert.KeyPair{interCert}
	}
	// Generate the peer and auth certs if not present
	var err error
	if s.PeerCert == nil {
		s.Log.Debugf("Generating peer cert")
		if s.PeerCert, err = cert.GenerateStandardKeyPair(s.IntermediateCACerts[0], nil, nil); err != nil {
			return nil, fmt.Errorf("failed generating peer cert: %w", err)
		}
	}
	if s.AuthCert == nil {
		s.Log.Debugf("Generating auth cert")
		if s.AuthCert, err = cert.GenerateStandardKeyPair(s.IntermediateCACerts[0], nil, nil); err != nil {
			return nil, fmt.Errorf("failed generating auth cert: %w", err)
		}
	}
	// Create TLS listener if not present
	if s.TLSListenNetwork == "" {
		s.TLSListenNetwork = "tcp"
	}
	if s.TLSListenAddr == "" {
		s.TLSListenAddr = ":0"
	}
	s.TLSListener = config.TLSListenerOverride
	if s.TLSListener == nil {
		cert, err := s.PeerCert.CreateTLSCertificate()
		if err != nil {
			return nil, fmt.Errorf("failed creating TLS cert from peer cert: %w", err)
		}
		s.Log.Debugf("Starting TLS listener on %v", s.TLSListenAddr)
		s.TLSListener, err = tls.Listen(s.TLSListenNetwork, s.TLSListenAddr,
			&tls.Config{Certificates: []tls.Certificate{cert}})
		if err != nil {
			return nil, fmt.Errorf("failed starting TLS listener: %w", err)
		}
	}
	// Get TLS addr
	addr, _ := s.TLSListener.Addr().(*net.TCPAddr)
	if addr == nil {
		return nil, fmt.Errorf("TLS listener is not TCP")
	}
	// Start mDNS server if not present
	if s.BroadcastInstanceName == "" {
		s.BroadcastInstanceName = "TakeCast"
	}
	if s.BroadcastFriendlyName == "" {
		s.BroadcastFriendlyName = "TakeCast"
	}
	if s.ID == "" {
		s.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	s.BroadcastServer = s.BroadcastServerOverride
	if s.BroadcastServer == nil {
		broadcastTextMap := map[string]string{
			"id": s.ID,
			"ve": "02",
			"md": "Chromecast",
			"fn": s.BroadcastFriendlyName,
			"ca": "5",
			"st": "0",
			"rs": "",
			"ic": "/setup/icon.png",
		}
		for k, v := range s.BroadcastTextOverrides {
			if v == "" {
				delete(broadcastTextMap, k)
			} else {
				broadcastTextMap[k] = v
			}
		}
		broadcastText := []string{}
		for k, v := range broadcastTextMap {
			broadcastText = append(broadcastText, k+"="+v)
		}
		s.Log.Debugf("Broadcasting mDNS for %v on port %v with TXT %v", s.BroadcastInstanceName, addr.Port, broadcastText)
		s.BroadcastServer, err = zeroconf.Register(s.BroadcastInstanceName, "_googlecast._tcp", "local.", addr.Port,
			broadcastText, s.BroadcastIfaces)
		if err != nil {
			return nil, fmt.Errorf("failed registering mDNS server: %w", err)
		}
	}
	// Set connection factory
	if s.NewConn == nil {
		s.NewConn = receiver.NewConn
	}
	// Create default receiver if no factory given
	if s.ReceiverForConn == nil {
		s.Receiver = receiver.NewReceiver(receiver.ReceiverConfig{Log: s.Log})
	}
	success = true
	return s, nil
}

// Blocks waiting for connection. Use Serve to call this repeatedly until close.
func (s *Server) Accept() (receiver.Conn, error) {
	s.Log.Debugf("Waiting for connection")
	netConn, err := s.TLSListener.Accept()
	if err != nil {
		return nil, err
	}
	return s.NewConn(receiver.ConnConfig{
		Socket:              netConn,
		IntermediateCACerts: s.IntermediateCACerts,
		PeerCert:            s.PeerCert,
		AuthCert:            s.AuthCert,
		Log:                 s.Log,
	})
}

// Runs until error or close. Basically just does Accept+ServeConn in a loop.
// Always returns error.
func (s *Server) Serve() error {
	for {
		conn, err := s.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err := s.ServeConn(conn); err == io.EOF {
				s.Log.Infof("Sender closed connection")
			} else if err == context.Canceled {
				s.Log.Infof("Receiver closed connection")
			} else {
				s.Log.Warnf("Connection failed: %v", err)
			}
		}()
	}
}

// Blocks and will close conn when done
func (s *Server) ServeConn(conn receiver.Conn) error {
	defer conn.Close()
	// Get receiver
	r := s.Receiver
	if r == nil {
		var err error
		if r, err = s.ReceiverForConn(conn); err != nil {
			return err
		}
	}
	// Connect channel
	ch, err := r.ConnectChannel(s.ctx, conn)
	if err != nil {
		return err
	}
	return ch.Run(s.ctx)
}

func (s *Server) Close() error {
	s.cancel()
	// Close servers if they are not overrides
	var lastErr error
	if s.Receiver != nil {
		s.Log.Debugf("Closing receiver")
		if err := s.Receiver.Close(); err != nil {
			lastErr = err
		}
		s.Receiver = nil
	}
	if s.BroadcastServerOverride == nil && s.BroadcastServer != nil {
		s.Log.Debugf("Closing mDNS server")
		s.BroadcastServer.Shutdown()
		s.BroadcastServer = nil
	}
	if s.TLSListenerOverride == nil && s.TLSListener != nil {
		s.Log.Debugf("Closing TLS listener")
		if err := s.TLSListener.Close(); err != nil {
			lastErr = err
		}
		s.TLSListener = nil
	}
	return lastErr
}
