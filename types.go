package spireauthlib

import (
	"github.com/sirupsen/logrus"
	delegated "github.com/spiffe/spire-api-sdk/proto/spire/api/agent/delegatedidentity/v1"
)

type SpiffeIDConfig struct {
	AuthorizedSpiffeIDs []string `yaml:"authorized_spiffe_ids"`
}

type ServerAuth struct {
	AllowedSpiffeIDsFile string `yaml:"allowed_spiffe_ids_file" ignore_on_empty:"true"`
	UdsPath              string `yaml:"uds_path" ignore_on_empty:"true"`
	Logger               *logrus.Logger
}

type ClientAuth struct {
	UdsPath    string `yaml:"uds_path" ignore_on_empty:"true"`
	ServerSvid string `yaml:"server_svid" ignore_on_empty:"true"`
	Logger     *logrus.Logger
}

type DelegatedAuth struct {
	UdsPath         string `yaml:"uds_path" ignore_on_empty:"true"`
	AdminUdsPath    string `yaml:"admin_uds_path" ignore_on_empty:"true"`
	DelegatedClient delegated.DelegatedIdentityClient
	Logger          *logrus.Logger
}
