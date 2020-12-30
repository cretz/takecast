package receiver

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/cretz/takecast/pkg/cert"
	"github.com/cretz/takecast/pkg/log"
	"github.com/cretz/takecast/pkg/receiver/cast_channel"
	"google.golang.org/protobuf/proto"
)

type Conn interface {
	Auth(*DeviceAuthRequestMessage) (*cast_channel.AuthResponse, error)
	Receive() (*cast_channel.CastMessage, error)
	// Source and destination are replaced with defaults if not present
	Send(*cast_channel.CastMessage) error
	Close() error
}

func receiveRequestMessage(conn Conn) (RequestMessage, error) {
	msg, err := conn.Receive()
	if err != nil {
		return nil, err
	}
	return UnmarshalRequestMessage(msg)
}

type conn struct {
	ConnConfig
}

type ConnConfig struct {
	Socket              io.ReadWriteCloser
	IntermediateCACerts []*cert.KeyPair
	PeerCert            *cert.KeyPair
	AuthCert            *cert.KeyPair
}

func (c *ConnConfig) Auth(d *DeviceAuthRequestMessage) (*cast_channel.AuthResponse, error) {
	if len(c.IntermediateCACerts) == 0 || c.PeerCert == nil || c.AuthCert == nil {
		return nil, fmt.Errorf("missing certificate info")
	}
	// Build auth response
	resp := &cast_channel.AuthResponse{
		ClientAuthCertificate: c.AuthCert.DERBytes,
		SignatureAlgorithm:    d.Challenge.SignatureAlgorithm,
		SenderNonce:           d.Challenge.SenderNonce,
		HashAlgorithm:         d.Challenge.HashAlgorithm,
	}
	for _, inter := range c.IntermediateCACerts {
		resp.IntermediateCertificate = append(resp.IntermediateCertificate, inter.DERBytes)
	}
	// Create hash
	var hash crypto.Hash
	switch d.Challenge.GetHashAlgorithm() {
	case cast_channel.HashAlgorithm_SHA1:
		hash = crypto.SHA1
	case cast_channel.HashAlgorithm_SHA256:
		hash = crypto.SHA256
	default:
		return nil, fmt.Errorf("unrecognized hash algorithm: %v", d.Challenge.GetHashAlgorithm())
	}
	toSign := make([]byte, 0, len(d.Challenge.SenderNonce)+len(c.PeerCert.DERBytes))
	toSign = append(toSign, d.Challenge.SenderNonce...)
	toSign = append(toSign, c.PeerCert.DERBytes...)
	hasher := hash.New()
	if _, err := hasher.Write(toSign); err != nil {
		return nil, fmt.Errorf("failed hashing: %w", err)
	}
	hashed := hasher.Sum(nil)
	// Do the signature
	var err error
	switch d.Challenge.GetSignatureAlgorithm() {
	case cast_channel.SignatureAlgorithm_RSASSA_PKCS1v15:
		resp.Signature, err = rsa.SignPKCS1v15(rand.Reader, c.AuthCert.PrivKey, hash, hashed)
		if err != nil {
			return nil, fmt.Errorf("failed signing: %w", err)
		}
	case cast_channel.SignatureAlgorithm_RSASSA_PSS:
		resp.Signature, err = rsa.SignPSS(rand.Reader, c.AuthCert.PrivKey, hash, hashed, nil)
		if err != nil {
			return nil, fmt.Errorf("failed signing: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown sig algo: %v", d.Challenge.GetSignatureAlgorithm())
	}
	return resp, nil
}

func NewConn(config ConnConfig) (Conn, error) {
	if config.Socket == nil {
		return nil, fmt.Errorf("missing socket")
	}
	return &conn{ConnConfig: config}, nil
}

func (c *conn) Receive() (*cast_channel.CastMessage, error) {
	// Get msg size
	byts := make([]byte, 4)
	if _, err := io.ReadFull(c.Socket, byts); err != nil {
		return nil, fmt.Errorf("failed reading size: %w", err)
	}
	msgSize := binary.BigEndian.Uint32(byts)
	// Get actual message
	byts = make([]byte, msgSize)
	if _, err := io.ReadFull(c.Socket, byts); err != nil {
		return nil, fmt.Errorf("failed reading message: %v", err)
	}
	var msg cast_channel.CastMessage
	if err := proto.Unmarshal(byts, &msg); err != nil {
		return nil, fmt.Errorf("failed unmarshaling msg: %v", err)
	}
	log.Debugf("Received message: %v", &msg)
	return &msg, nil
}

var defaultSourceID = "receiver-0"
var defaultDestinationID = "sender-0"

func (c *conn) Send(msg *cast_channel.CastMessage) error {
	// Set default source and destination
	if msg.SourceId == nil {
		msg.SourceId = &defaultSourceID
	}
	if msg.DestinationId == nil {
		msg.DestinationId = &defaultDestinationID
	}
	log.Debugf("Sending message: %v", msg)
	byts, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed marshaling cast message: %w", err)
	}
	sizeByts := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeByts, uint32(len(byts)))
	if _, err = c.Socket.Write(sizeByts); err != nil {
		return fmt.Errorf("failed writing size: %w", err)
	}
	if _, err = c.Socket.Write(byts); err != nil {
		return fmt.Errorf("failed writing bytes: %w", err)
	}
	return nil
}

func (c *conn) Close() error { return c.Socket.Close() }
