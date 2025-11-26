package spireauthlib

import (
	"context"
	"fmt"
	"strings"

	delegated "github.com/spiffe/spire-api-sdk/proto/spire/api/agent/delegatedidentity/v1"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (d *DelegatedAuth) GetDelegatedJWT(ctx context.Context, selectors []*types.Selector, audience string) (*delegated.FetchJWTSVIDsResponse, error) {
	adminPath := adminSocketPath
	if d.AdminUdsPath != "" {
		d.Logger.Infof("Using admin UDS socket path override from config")
		adminPath = d.AdminUdsPath
	}
	if adminPath != "" && !strings.HasPrefix(adminPath, "unix:") {
		adminPath = "unix://" + adminPath
		d.Logger.Infof("Using admin UDS socket path %s", adminPath)
	}

	dlgApiConn, err := grpc.NewClient(adminPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dlgClient := delegated.NewDelegatedIdentityClient(dlgApiConn)
	defer dlgApiConn.Close()

	JwtSvidReq := delegated.FetchJWTSVIDsRequest{
		Selectors: selectors,
		Audience:  []string{audience},
	}

	d.Logger.Infof("Unmarshaled delegated JWT SVID request for selectors:%s audience:%s", JwtSvidReq.Selectors, JwtSvidReq.Audience)

	JwtSvidResp, err := dlgClient.FetchJWTSVIDs(ctx, &JwtSvidReq)

	if err != nil {
		return nil, fmt.Errorf("unable to fetch JWT SVIDs via delegated identity: %w", err)
	}

	return JwtSvidResp, nil
}
