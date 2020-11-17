/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package openstack

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
	"k8s.io/klog"
)

func NewClientFactoryFromSecret(secret *corev1.Secret) (ServiceClientFactory, error) {
	creds, err := extractCredentials(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("error extracting credentials from secret: %v", err)
	}
	return newClientFactoryFromCredentials(creds)
}

func newClientFactoryFromCredentials(credentials *credentials) (ServiceClientFactory, error){
	config := &tls.Config{}

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
		config.BuildNameToCertificate()
	}

	clientOpts := new(clientconfig.ClientOpts)
	authInfo := &clientconfig.AuthInfo{
		AuthURL:        credentials.AuthURL,
		Username:       credentials.Username,
		Password:       credentials.Password,
		DomainName:     credentials.DomainName,
		DomainID:       credentials.DomainID,
		ProjectName:    credentials.TenantName,
		ProjectID:      credentials.TenantID,
		UserDomainName: credentials.UserDomainName,
		UserDomainID:   credentials.UserDomainID,
	}
	clientOpts.AuthInfo = authInfo

	ao, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create client auth options: %v", err)
	}

	provider, err :=  openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated client: %v", err)
	}

	// Set UserAgent
	provider.UserAgent.Prepend("Machine Controller Provider Openstack")

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient = http.Client{
		Transport: transport,
	}

	if klog.V(6) {
		provider.HTTPClient.Transport = &client.RoundTripper{
			Rt:     provider.HTTPClient.Transport,
			Logger: &logger{},
		}
	}

	err = openstack.Authenticate(provider, *ao)
	if err != nil {
		return nil, err
	}

	return &ServiceClientFactoryImpl{
		providerClient: provider,
	}, nil
}

// IsNotFoundError checks if an error returned by OpenStack is caused by HTTP 404 status code.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(gophercloud.ErrDefault404); ok {
		return true
	}

	if _, ok := err.(gophercloud.Err404er); ok {
		return true
	}

	return false
}

type logger struct{}

func (l logger) Printf(format string, args ...interface{}) {
	// extra check in case, when verbosity has been changed dynamically
	if klog.V(6) {
		var skip int
		var found bool
		var gc = "/github.com/gophercloud/gophercloud"

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