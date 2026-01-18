package spireauthlib

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spiffe/go-spiffe/v2/bundle/jwtbundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// GetTlsClient creates a TLS-enabled HTTP client using SPIFFE mTLS.

func (c *ClientAuth) GetTlsClient(ctx context.Context) (*http.Client, error) {
	udsPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	// Override with config value if set
	if c.UdsPath != "" {
		c.Logger.Infof("Using UDS socket path override from config")
		udsPath = c.UdsPath
	}

	if udsPath != "" && !strings.HasPrefix(udsPath, "unix:") {
		udsPath = "unix://" + udsPath
		c.Logger.Infof("Using UDS socket path %s", udsPath)
	}

	if udsPath == "" {
		udsPath = "unix:///tmp/agent.sock"
		c.Logger.Infof("Using default UDS socket endpoint: %s", udsPath)
	}

	mySvid, err := workloadapi.FetchX509SVID(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch X509 SVID: %w", err)
	}
	myTD := mySvid.ID.TrustDomain()

	c.Logger.Infof("Workload trust domain: %s", myTD)
	c.Logger.Infof("Workload SVID: %s", mySvid.ID.URL())

	// Create a `workloadapi.X509Source`, it will connect to Workload API using provided socket path
	// If socket path is not defined using `workloadapi.SourceOption`, value from environment variable `SPIFFE_ENDPOINT_SOCKET` is used.
	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(udsPath)))
	if err != nil {
		return nil, fmt.Errorf("unable to create X509Source: %w", err)
	}
	// If ServerSvid is set, parse it
	var serverID spiffeid.ID
	if c.ServerSvid != "" && c.ServerSvid != "any" {
		serverID, err = spiffeid.FromString(c.ServerSvid)
		if err != nil {
			return nil, fmt.Errorf("unable to parse server SVID SPIFFE ID: %w", err)
		}
	}
	var tlsConfig *tls.Config
	if serverID != (spiffeid.ID{}) {
		c.Logger.Infof("Authorizing connection to server SVID: %s", serverID.URL())
		// Allow connection only to the specified server SPIFFE ID
		tlsConfig = tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeID(serverID))
	} else if c.ServerSvid == "any" {
		// Allow connection to any server
		c.Logger.Infof("Authorizing connection to any server")
		tlsConfig = tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	} else {
		// Allow connection to all my trust domain member servers by default
		c.Logger.Infof("Authorizing connection to any server in trust domain: %s", myTD)
		tlsConfig = tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeMemberOf(myTD))
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client, nil
}

func (c *ClientAuth) GetJWT(ctx context.Context, audience string) (*jwtbundle.Set, *jwtsvid.SVID, error) {
	udsPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	// Override with config value if set
	if c.UdsPath != "" {
		c.Logger.Infof("Using UDS socket path override from config")
		udsPath = c.UdsPath
	}

	if udsPath != "" && !strings.HasPrefix(udsPath, "unix:") {
		udsPath = "unix://" + udsPath
		c.Logger.Infof("Using UDS socket path %s", udsPath)
	}

	if udsPath == "" {
		udsPath = "unix:///tmp/agent.sock"
		c.Logger.Infof("Using default UDS socket endpoint: %s", udsPath)
	}

	mysvid, err := workloadapi.FetchX509SVID(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch X509 SVID: %w", err)
	}
	c.Logger.Infof("Workload SVID: %s", mysvid.ID.URL())
	jwtParams := jwtsvid.Params{
		Audience: audience,
		Subject:  mysvid.ID,
	}
	myjwt, err := workloadapi.FetchJWTSVID(ctx, jwtParams)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch JWT SVID: %w", err)
	}

	jwtbundle, err := workloadapi.FetchJWTBundles(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch JWT bundles: %w", err)
	}
	c.Logger.Infof("JWT Bundle for trust domain %s has %d keys", myjwt.ID.TrustDomain(), jwtbundle.Len())
	c.Logger.Infof("JWT SVID: %s", myjwt.ID.URL())
	return jwtbundle, myjwt, nil
}

func (c *ClientAuth) ValidateJWT(jwtBundle *jwtbundle.Set, jwtSvid *jwtsvid.SVID) error {
	svid, err := jwtsvid.ParseAndValidate(jwtSvid.Marshal(), jwtBundle, jwtSvid.Audience)
	if err != nil {
		return fmt.Errorf("unable to validate JWT SVID: %w", err)
	}
	c.Logger.Infof("Successfully validated JWT SVID: %s", svid.ID.URL())
	return nil
}
