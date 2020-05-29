package datasources

const (
	padL string = "^L^"
	padR string = "^R^"
)

// ConfInterface defines the interface for config structs
type ConfInterface interface {
	Init()
}

// ConfBase is the common type for all modules
//
// Custom modules should respect these options
type ConfBase struct {
	// Override global setting
	WarnOnly *bool `yaml:"warnings_only,omitempty"`
	// 2-element array defining padding for header (title)
	PadHeader []int `yaml:"pad_header,flow"`
	// 2-element array defining padding for content (details)
	PadContent []int `yaml:"pad_content,flow"`
}

// Init sets `PadHeader` and `PadContent` to [0, 0]
func (c *ConfBase) Init() {
	c.PadHeader = []int{0, 0}
	c.PadContent = []int{1, 0}
}

// ConfBaseWarn extends ConfBase with warning and critical values
type ConfBaseWarn struct {
	ConfBase `yaml:",inline"`
	Warn     int `yaml:"warn"`
	Crit     int `yaml:"crit"`
}

// Init sets warning to 70 and critical to 90
func (c *ConfBaseWarn) Init() {
	c.ConfBase.Init()
	c.Warn = 70
	c.Crit = 90
}
