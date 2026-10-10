package factory

type config struct {
	Workers          int    `mapstructure:"workers"`
	Timeout          string `mapstructure:"timeout"`
	DiscoveryTimeout string `mapstructure:"discovery_timeout"`
	Debug            bool   `mapstructure:"debug"`
}

var Cfg config
