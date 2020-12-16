// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
)

type Credentials struct {
	DomainName     string
	DomainID       string
	UserDomainName string
	UserDomainID   string

	TenantID   string
	TenantName string

	Username string

	CACert     []byte
	ClientKey  []byte
	ClientCert []byte
	Insecure   bool

	Password string
	AuthURL  string
}

func ExtractCredentials(secret *corev1.Secret) *Credentials {
	data := secret.Data

	authURL := data[cloudprovider.OpenStackAuthURL]
	username := data[cloudprovider.OpenStackUsername]
	password := data[cloudprovider.OpenStackPassword]

	// optional OS_USER_DOMAIN_NAME
	userDomainName := data[cloudprovider.OpenStackUserDomainName]
	// optional OS_USER_DOMAIN_ID
	userDomainID := data[cloudprovider.OpenStackUserDomainID]

	domainName := data[cloudprovider.OpenStackDomainName]
	domainID := data[cloudprovider.OpenStackDomainID]

	tenantName := data[cloudprovider.OpenStackTenantName]
	tenantID := data[cloudprovider.OpenStackTenantID]

	var caCert, clientCert, clientKey []byte
	var ok bool
	if caCert, ok = data[cloudprovider.OpenStackCACert]; !ok {
		caCert = nil
	}
	if clientCert, ok = data[cloudprovider.OpenStackClientCert]; !ok {
		clientCert = nil
	}
	if clientKey, ok = data[cloudprovider.OpenStackClientKey]; !ok {
		clientKey = nil
	}

	insecure := strings.TrimSpace(string(data[cloudprovider.OpenStackInsecure])) == "true"

	return &Credentials{
		DomainName:     strings.TrimSpace(string(domainName)),
		DomainID:       strings.TrimSpace(string(domainID)),
		UserDomainName: strings.TrimSpace(string(userDomainName)),
		UserDomainID:   strings.TrimSpace(string(userDomainID)),
		TenantName:     strings.TrimSpace(string(tenantName)),
		TenantID:       strings.TrimSpace(string(tenantID)),
		Username:       strings.TrimSpace(string(username)),
		Password:       strings.TrimSpace(string(password)),
		AuthURL:        strings.TrimSpace(string(authURL)),
		ClientCert:     clientCert,
		ClientKey:      clientKey,
		CACert:         caCert,
		Insecure:       insecure,
	}
}
