package options

// ProxyRunOptions specifies the parameters to run ssh proxy.
type ProxyRunOptions struct {
	Username   string
	Password   string
	SSHKeyFile string

	BindAddress       string
	Port              int
	TLSCertFile       string
	TLSPrivateKeyFile string
}

// NewProxyRunOptions provides the default parameters to run ssh proxy.
func NewProxyRunOptions() *ProxyRunOptions {
	return &ProxyRunOptions{}
}
