package env

import "log/slog"

type Mode string

const (
	Test  Mode = "test"
	Local Mode = "local"
	Dev   Mode = "dev"
	Prod  Mode = "prod"
)

var currentMode = Test

func SetMode(mode Mode) {
	if !mode.Validate() {
		panic("invalid mode: " + mode.String())
	}
	currentMode = mode
}

func Current() Mode {
	return currentMode
}

func (e Mode) String() string {
	return string(e)
}

func (e Mode) Validate() bool {
	switch e {
	case Local, Test, Dev, Prod:
		return true
	default:
		return false
	}
}

func (e Mode) SlogLevel() slog.Level {
	switch e {
	case Test, Local, Dev:
		return slog.LevelDebug
	case Prod:
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
