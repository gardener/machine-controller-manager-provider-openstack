// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"strings"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
)

type credentials struct {
	DomainName     string
	DomainID       string
	UserDomainName string
	UserDomainID   string

	TenantID   string
	TenantName string

	Username string
	Password string

	ApplicationCredentialID     string
	ApplicationCredentialName   string
	ApplicationCredentialSecret string

	CACert     []byte
	ClientKey  []byte
	ClientCert []byte
	Insecure   bool

	AuthURL string
}

func extractCredentialsFromSecretData(data map[string][]byte) *credentials {
	if data == nil {
		return nil
	}

	authURL := data[cloudprovider.OpenStackAuthURL]
	username := data[cloudprovider.OpenStackUsername]
	password := data[cloudprovider.OpenStackPassword]

	applicationCredentialID := data[cloudprovider.OpenStackApplicationCredentialID]
	applicationCredentialName := data[cloudprovider.OpenStackApplicationCredentialName]
	applicationCredentialSecret := data[cloudprovider.OpenStackApplicationCredentialSecret]

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

	return &credentials{
		DomainName:                  strings.TrimSpace(string(domainName)),
		DomainID:                    strings.TrimSpace(string(domainID)),
		UserDomainName:              strings.TrimSpace(string(userDomainName)),
		UserDomainID:                strings.TrimSpace(string(userDomainID)),
		TenantName:                  strings.TrimSpace(string(tenantName)),
		TenantID:                    strings.TrimSpace(string(tenantID)),
		Username:                    strings.TrimSpace(string(username)),
		Password:                    strings.TrimSpace(string(password)),
		ApplicationCredentialID:     strings.TrimSpace(string(applicationCredentialID)),
		ApplicationCredentialName:   strings.TrimSpace(string(applicationCredentialName)),
		ApplicationCredentialSecret: strings.TrimSpace(string(applicationCredentialSecret)),
		AuthURL:                     strings.TrimSpace(string(authURL)),
		ClientCert:                  clientCert,
		ClientKey:                   clientKey,
		CACert:                      caCert,
		Insecure:                    insecure,
	}
}
