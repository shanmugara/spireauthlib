package spireauthlib

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
	delegated "github.com/spiffe/spire-api-sdk/proto/spire/api/agent/delegatedidentity/v1"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (d *DelegatedAuth) GetDelegatedJWT(ctx context.Context, ns string, sa string) (*tls.Config, error) {
	udsPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	// Override with config value if set
	if d.UdsPath != "" {
		d.Logger.Infof("Using UDS socket path override from config")
		udsPath = d.UdsPath
	}

	if udsPath != "" && !strings.HasPrefix(udsPath, "unix:") {
		udsPath = "unix://" + udsPath
		d.Logger.Infof("Using UDS socket path %s", udsPath)
	}

	if udsPath == "" {
		udsPath = "unix:///tmp/agent.sock"
		d.Logger.Infof("Using default UDS socket endpoint: %s", udsPath)
	}

	src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(udsPath)))
	if err != nil {
		return nil, fmt.Errorf("unable to create X509Source: %w", err)
	}
	defer src.Close()

	dlgApiConn, err := grpc.NewClient(adminSocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dlgClient := delegated.NewDelegatedIdentityClient(dlgApiConn)

	JwtSvidReq := delegated.FetchJWTSVIDsRequest{
		Selectors: []*types.Selector{
			{Type: "k8s", Value: "ns:" + ns},
			{Type: "k8s", Value: "sa:" + sa},
		},
		Audience: []string{"omegahome"},
	}

	JwtSvidResp, err := dlgClient.FetchJWTSVIDs(ctx, &JwtSvidReq)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch JWT SVIDs via delegated identity: %w", err)
	}
	for _, s := range JwtSvidResp.Svids {
		d.Logger.Infof("Delegated SVID: %s JWT: %s", s.Id.Path, s.Token)
	}

	return nil, nil
}
