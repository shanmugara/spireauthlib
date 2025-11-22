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

	adminPath := adminSocketPath
	if d.AdminUdsPath != "" {
		d.Logger.Infof("Using admin UDS socket path override from config")
		adminPath = d.AdminUdsPath
	}
	if adminPath != "" && !strings.HasPrefix(adminPath, "unix:") {
		adminPath = "unix://" + adminPath
		d.Logger.Infof("Using admin UDS socket path %s", adminPath)
	}

	// Dial the admin socket using gRPC. Use DialContext so we honor the provided ctx.
	//dlgApiConn, err := grpc.DialContext(ctx, adminPath, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dlgApiConn, err := grpc.NewClient(adminPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dlgClient := delegated.NewDelegatedIdentityClient(dlgApiConn)
	defer dlgApiConn.Close()

	JwtSvidReq := delegated.FetchJWTSVIDsRequest{
		Selectors: []*types.Selector{
			{Type: "k8s", Value: "ns:" + ns},
			{Type: "k8s", Value: "sa:" + sa},
		},
		Audience: []string{"omegahome"},
	}
	d.Logger.Infof("Unmarshaled delegated JWT SVID request for selectors:%s audience:%s", JwtSvidReq.Selectors, JwtSvidReq.Audience)

	JwtSvidResp, err := dlgClient.FetchJWTSVIDs(ctx, &JwtSvidReq)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch JWT SVIDs via delegated identity: %w", err)
	}
	for _, s := range JwtSvidResp.Svids {
		d.Logger.Infof("Delegated SVID: %s JWT: %s", s.Id.Path, s.Token)
	}

	return nil, nil
}
