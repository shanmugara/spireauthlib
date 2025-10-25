package spireauthlib

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"gopkg.in/yaml.v3"
)

const (
	udsSocketPath = "/tmp/spire-agent/public/api.sock"
)

var Logger *logrus.Logger

// SpiffeIDConfig reads from the YAML file containing authorized Spiffe IDs
//and returns them as a slice of spiffeid.ID

func (s *ServerAuth) LoadSpiffeIDs() ([]spiffeid.ID, error) {
	var cfg SpiffeIDConfig
	data, err := os.ReadFile(s.AllowedSpiffeIDsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SPIFFE IDs from %s: %w", s.AllowedSpiffeIDsFile, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	var spiffeIds []spiffeid.ID
	for _, idStr := range cfg.AuthorizedSpiffeIDs {
		id, err := spiffeid.FromString(idStr)
		if err != nil {
			return nil, err
		}
		spiffeIds = append(spiffeIds, id)
	}
	return spiffeIds, nil
}

func (s *ServerAuth) getMySvid(ctx context.Context) (spiffeid.ID, error) {
	svid, err := workloadapi.FetchX509SVID(ctx)
	if err != nil {
		return spiffeid.ID{}, fmt.Errorf("unable to fetch X509 SVID: %w", err)
	}
	return svid.ID, nil
}

// GetTlsConfig creates a TLS configuration for a server using SPIFFE mTLS.

func (s ServerAuth) GetTlsConfig(ctx context.Context) (*tls.Config, error) {
	// prefer package logger if set
	if Logger == nil {
		Logger = logrus.New()
	}

	wlSvid, err := s.getMySvid(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get my SVID: %w", err)
	}

	allowed, err := s.LoadSpiffeIDs()
	if err != nil {
		Logger.Errorf("unable to load allowed SPIFFE IDs: %v", err)
		allowed = []spiffeid.ID{}
	}

	// Choose socket address: prefer SPIFFE_ENDPOINT_SOCKET env, otherwise fall back to default
	udsPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if udsPath == "" {
		Logger.Infof("SPIFFE_ENDPOINT_SOCKET not set")
		udsPath = udsSocketPath
	} else {
		Logger.Infof("using SPIFFE_ENDPOINT_SOCKET: %s", udsPath)
	}

	// If a plain filesystem path was provided, prefix with unix:// to form a valid URI
	if !strings.Contains(udsPath, "://") {
		udsPath = "unix://" + udsPath
	}

	// Create a `workloadapi.X509Source`, it will connect to Workload API using provided socket.
	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(udsPath)))
	if err != nil {
		return nil, fmt.Errorf("unable to create X509Source: %w", err)
	}

	var tlsConfig *tls.Config
	if len(allowed) > 0 {
		// Allow only the specified SPIFFE IDs
		Logger.Infof("using %d allowed X509 SVID(s) for authorization", len(allowed))
		tlsConfig = tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeOneOf(allowed...))
	} else {
		// Allow any workload from the default trust domain
		Logger.Warn("no allowed X509 SVID, using default trust domain authorization: " + wlSvid.TrustDomain().String())
		tlsConfig = tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeMemberOf(wlSvid.TrustDomain()))
	}

	return tlsConfig, nil
}
