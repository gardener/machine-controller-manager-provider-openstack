// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Factory can create clients for Nova and Neutron OpenStack services.
type Factory struct {
	providerClient *gophercloud.ProviderClient
}

// Option can modify client parameters by manipulating EndpointOpts.
type Option func(opts gophercloud.EndpointOpts) gophercloud.EndpointOpts

// NewFactoryFromSecretData can create a Factory from the a kubernetes secret's data.
func NewFactoryFromSecretData(ctx context.Context, data map[string][]byte) (*Factory, error) {
	if data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	creds := extractCredentialsFromSecretData(data)
	provider, err := newAuthenticatedProviderClientFromCredentials(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("error creating OpenStack client from credentials: %w", err)
	}

	return &Factory{
		providerClient: provider,
	}, nil
}

// NewFactoryFromSecret can create a Factory from the a kubernetes secret.
func NewFactoryFromSecret(ctx context.Context, secret *corev1.Secret) (*Factory, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret cannot be nil")
	}

	return NewFactoryFromSecretData(ctx, secret.Data)
}

func newAuthenticatedProviderClientFromCredentials(ctx context.Context, credentials *credentials) (*gophercloud.ProviderClient, error) {
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: credentials.AuthURL,
		// AllowReauth should be set to true if you grant permission for Gophercloud to
		// cache your credentials in memory, and to allow Gophercloud to attempt to
		// re-authenticate automatically if/when your token expires.
		AllowReauth: true,
	}

	if credentials.ApplicationCredentialID != "" {
		authOpts.ApplicationCredentialID = credentials.ApplicationCredentialID
		authOpts.ApplicationCredentialName = credentials.ApplicationCredentialName
		authOpts.ApplicationCredentialSecret = credentials.ApplicationCredentialSecret
	} else {
		authOpts.Username = credentials.Username
		authOpts.Password = credentials.Password
		authOpts.DomainName = credentials.DomainName
		authOpts.TenantName = credentials.TenantName
	}

	tlsConfig := &tls.Config{} // #nosec: G402 -- Can be parameterized.
	if credentials.CACert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(credentials.CACert)
		tlsConfig.RootCAs = caCertPool
	}
	if credentials.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}
	if credentials.ClientCert != nil {
		cert, err := tls.X509KeyPair(credentials.ClientCert, credentials.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create X509 key pair: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	httpClient := http.Client{
		Transport: transport,
	}

	provider, err := config.NewProviderClient(
		ctx,
		authOpts,
		config.WithTLSConfig(tlsConfig),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider client: %w", err)
	}

	provider.UserAgent.Prepend("Machine Controller Provider Openstack")

	if klog.V(6).Enabled() {
		provider.HTTPClient.Transport = &loggingRoundTripper{
			Rt:     provider.HTTPClient.Transport,
			Logger: &logger{},
		}
	}

	return provider, nil
}

type logger struct{}

func (l logger) Printf(format string, args ...interface{}) {
	// extra check in case, when verbosity has been changed dynamically
	if klog.V(6).Enabled() {
		var skip int
		var found bool
		gc := "/github.com/gophercloud/gophercloud"

		// detect the depth of the actual function, which calls gophercloud code
		// 10 is the common depth from the logger to "github.com/gophercloud/gophercloud"
		for i := 10; i <= 20; i++ {
			if _, file, _, ok := runtime.Caller(i); ok && !found && strings.Contains(file, gc) {
				found = true
				continue
			} else if ok && found && !strings.Contains(file, gc) {
				skip = i
				break
			} else if !ok {
				break
			}
		}

		for _, v := range strings.Split(fmt.Sprintf(format, args...), "\n") {
			klog.InfoDepth(skip, v)
		}
	}
}

// WithRegion returns an Option that can modify the region a client targets.
func WithRegion(region string) Option {
	return func(opts gophercloud.EndpointOpts) gophercloud.EndpointOpts {
		opts.Region = region
		return opts
	}
}

// Compute returns a client for OpenStack's Nova service.
func (f *Factory) Compute(opts ...Option) (Compute, error) {
	eo := gophercloud.EndpointOpts{}
	for _, opt := range opts {
		eo = opt(eo)
	}

	return newNovaV2(f.providerClient, eo)
}

// Network returns a client for OpenStack's Neutron service.
func (f *Factory) Network(opts ...Option) (Network, error) {
	eo := gophercloud.EndpointOpts{}
	for _, opt := range opts {
		eo = opt(eo)
	}

	return newNeutronV2(f.providerClient, eo)
}

// Storage returns a client for OpenStack's Cinder service.
func (f *Factory) Storage(opts ...Option) (Storage, error) {
	eo := gophercloud.EndpointOpts{}
	for _, opt := range opts {
		eo = opt(eo)
	}

	return newCinderV3(f.providerClient, eo)
}
