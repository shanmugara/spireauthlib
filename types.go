package spireauthlib

type SpiffeIDConfig struct {
	AuthorizedSpiffeIDs []string `yaml:"authorized_spiffe_ids"`
}

type ServerAuth struct {
	AllowedSpiffeIDsFile string `yaml:"allowed_spiffe_ids_file" ignore_on_empty:"true"`
	UdsPath              string `yaml:"uds_path" ignore_on_empty:"true"`
}

type ClientAuth struct {
	UdsPath    string `yaml:"uds_path" ignore_on_empty:"true"`
	ServerSvid string `yaml:"server_svid" ignore_on_empty:"true"`
}
