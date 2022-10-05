package config

type Config struct {
	Background bool
	Windows    []Window
}

type Window struct {
	Main string
	Sub  string
}
