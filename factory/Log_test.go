package factory

import (
	"testing"
)

// create a unit for the log.go,InitLogger set debug to true in the argument
func TestLog(t *testing.T) {
	InitLogger(true)
	InitLogger(false)
}

// create a unit test for the viper.go InitConfig function
func TestViper(t *testing.T) {
	InitConfig()
}
