package factory

type config struct {
	Defaultworkersize int
	Timeout           string
	Isdebug           bool
	Lang              string
}

var Cfg config
