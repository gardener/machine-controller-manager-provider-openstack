// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/client"
	"github.com/gophercloud/utils/openstack/clientconfig"
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
func NewFactoryFromSecretData(data map[string][]byte) (*Factory, error) {
	if data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	creds := extractCredentialsFromSecretData(data)
	provider, err := newAuthenticatedProviderClientFromCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("error creating OpenStack client from credentials: %w", err)
	}

	return &Factory{
		providerClient: provider,
	}, nil
}

// NewFactoryFromSecret can create a Factory from the a kubernetes secret.
func NewFactoryFromSecret(secret *corev1.Secret) (*Factory, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret cannot be nil")
	}

	return NewFactoryFromSecretData(secret.Data)
}

func newAuthenticatedProviderClientFromCredentials(credentials *credentials) (*gophercloud.ProviderClient, error) {
	config := &tls.Config{} // #nosec: G402 -- Can be parameterized.

	if credentials.CACert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(credentials.CACert)
		config.RootCAs = caCertPool
	}

	if credentials.Insecure {
		config.InsecureSkipVerify = true
	}

	if credentials.ClientCert != nil {
		cert, err := tls.X509KeyPair(credentials.ClientCert, credentials.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create X509 key pair: %v", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	clientOpts := new(clientconfig.ClientOpts)
	authInfo := &clientconfig.AuthInfo{
		AuthURL:                     credentials.AuthURL,
		Username:                    credentials.Username,
		Password:                    credentials.Password,
		DomainName:                  credentials.DomainName,
		DomainID:                    credentials.DomainID,
		ProjectName:                 credentials.TenantName,
		ProjectID:                   credentials.TenantID,
		UserDomainName:              credentials.UserDomainName,
		UserDomainID:                credentials.UserDomainID,
		ApplicationCredentialID:     credentials.ApplicationCredentialID,
		ApplicationCredentialName:   credentials.ApplicationCredentialName,
		ApplicationCredentialSecret: credentials.ApplicationCredentialSecret,
	}
	clientOpts.AuthInfo = authInfo

	if clientOpts.AuthInfo.ApplicationCredentialSecret != "" {
		clientOpts.AuthType = clientconfig.AuthV3ApplicationCredential
	}

	ao, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create client auth options: %w", err)
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated client: %w", err)
	}

	// Set UserAgent
	provider.UserAgent.Prepend("Machine Controller Provider Openstack")

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient = http.Client{
		Transport: transport,
	}

	if klog.V(6).Enabled() {
		provider.HTTPClient.Transport = &client.RoundTripper{
			Rt:     provider.HTTPClient.Transport,
			Logger: &logger{},
		}
	}

	err = openstack.Authenticate(provider, *ao)
	if err != nil {
		return nil, err
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
