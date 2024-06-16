package config

var defaultConfig = Config{
	View:    "",
	Streams: []Stream{},
	Views:   []View{},
}

type Config struct {
	View    string   `json:"view"`
	Streams []Stream `json:"streams"`
	Views   []View   `json:"views"`
}

type Stream struct {
	UUID  string   `json:"uuid"`
	Main  string   `json:"main"`
	Sub   string   `json:"sub"`
	Flags []string `json:"flags"`
}

type View struct {
	UUID    string   `json:"uuid"`
	Type    string   `json:"type"` // [auto, view]
	Panes   []Pane   `json:"panes"`
	Streams []string `json:"streams"`
}

type Pane struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}
