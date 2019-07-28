package app

import (
	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/klog"

	"github.com/zhouhaibing089/sshproxy/cmd/sshproxy/app/options"
	"github.com/zhouhaibing089/sshproxy/pkg/version"
)

//NewProxyCommand creates a *cobra.Command object with default parameters
func NewProxyCommand() *cobra.Command {
	s := options.NewProxyRunOptions()
	cmd := &cobra.Command{
		Use:  "sshproxy",
		Long: `proxy ssh session via websocket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// set default options
			completedOptions, err := Complete(s)
			if err != nil {
				return err
			}
			// validate options
			if errs := completedOptions.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}
			return Run(completedOptions, genericapiserver.SetupSignalHandler())
		},
	}

	return cmd
}

// Run runs the ssh proxy server. This should never exit.
func Run(completedOptions completedProxyRunOptions, stopCh <-chan struct{}) error {
	// To help debugging, immediately log version
	klog.Infof("Version: %s", version.Get())

	server, err := CreateProxyServer(completedOptions, stopCh)
	if err != nil {
		return err
	}

	return server.Run(stopCh)
}

// completeProxyRunOptions is a private wrapper that enforces a call of Complete()
// before Run can be invoked.
type completedProxyRunOptions struct {
	*options.ProxyRunOptions
}

// Complete set default ProxyRunOptions.
// Should be called after sshproxy flags parsed.
func Complete(s *options.ProxyRunOptions) (completedProxyRunOptions, error) {
	var options completedProxyRunOptions
	options.ProxyRunOptions = s
	return options, nil
}

// CreateProxyServer instantiate a new proxy server.
func CreateProxyServer(completedOptions completedProxyRunOptions, stopCh <-chan struct{}) (*ProxyServer, error) {

	return nil, nil
}

// ProxyServer is a http server which proxies ssh interactions.
type ProxyServer struct {
}

// Run runs the proxy server.
func (p *ProxyServer) Run(stopCh <-chan struct{}) error {
	return nil
}
