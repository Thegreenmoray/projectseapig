package factory

type config struct {
	Workers int    `mapstructure:"workers"`
	Timeout string `mapstructure:"timeout"`
	Debug   bool   `mapstructure:"debug"`
}

var Cfg config
