package datasources

const (
	padL string = "$"
	padR string = "%"
)

// CommonConf is the common type for all modules
//
// Custom modules should respect these options
type CommonConf struct {
	FailedOnly *bool `yaml:"failedOnly,omitempty"`
	Header     []int `yaml:"header"`
	Content    []int `yaml:"content"`
}

// Init sets `Header` and `Content` to [0, 0]
func (c *CommonConf) Init() {
	var defPad = []int{0, 0}
	c.Content = defPad
	c.Header = defPad
}

// CommonWithWarnConf extends CommonConf with warning and critical values
type CommonWithWarnConf struct {
	CommonConf `yaml:",inline"`
	Warn       int `yaml:"warn"`
	Crit       int `yaml:"crit"`
}
