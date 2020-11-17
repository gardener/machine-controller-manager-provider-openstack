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
	"fmt"
	"strings"
)

const (
	// OpenStackAuthURL is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackAuthURL string = "authURL"
	// OpenStackCACert is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackCACert string = "caCert"
	// OpenStackInsecure is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackInsecure string = "insecure"
	// OpenStackDomainName is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackDomainName string = "domainName"
	// OpenStackDomainID is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackDomainID string = "domainID"
	// OpenStackTenantName is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackTenantName string = "tenantName"
	// OpenStackTenantID is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackTenantID string = "tenantID"
	// OpenStackUserDomainName is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackUserDomainName string = "userDomainName"
	// OpenStackUserDomainID is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackUserDomainID string = "userDomainID"
	// OpenStackUsername is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackUsername string = "username"
	// OpenStackPassword is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackPassword string = "password"
	// OpenStackClientCert is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackClientCert string = "clientCert"
	// OpenStackClientKey is a constant for a key name that is part of the OpenStack cloud credentials.
	OpenStackClientKey string = "clientKey"
)

type credentials struct {
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

func extractCredentials(data map[string][]byte) (*credentials, error) {
	authURL, ok := data[OpenStackAuthURL]
	if !ok {
		return nil, fmt.Errorf("missing %s in secret", OpenStackAuthURL)
	}
	username, ok := data[OpenStackUsername]
	if !ok {
		return nil, fmt.Errorf("missing %s in secret", OpenStackUsername)
	}
	password, ok := data[OpenStackPassword]
	if !ok {
		return nil, fmt.Errorf("missing %s in secret", OpenStackPassword)
	}

	// optional OS_USER_DOMAIN_NAME
	userDomainName := data[OpenStackUserDomainName]
	// optional OS_USER_DOMAIN_ID
	userDomainID := data[OpenStackUserDomainID]

	domainName, ok := data[OpenStackDomainName]
	domainID, ok2 := data[OpenStackDomainID]
	if !ok && !ok2 {
		return nil, fmt.Errorf("missing %s or %s in secret", OpenStackDomainName, OpenStackDomainID)
	}

	tenantName, ok := data[OpenStackTenantName]
	tenantID, ok2 := data[OpenStackTenantID]
	if !ok && !ok2 {
		return nil, fmt.Errorf("missing %s or %s in secret", OpenStackTenantName, OpenStackTenantID)
	}

	var caCert, clientCert, clientKey []byte
	if caCert, ok = data[OpenStackCACert]; !ok {
		caCert = nil
	}
	if clientCert, ok = data[OpenStackClientCert]; !ok {
		clientCert = nil
	}
	if clientKey, ok = data[OpenStackClientKey]; !ok {
		clientKey = nil
	}

	if len(clientCert) != 0 && len(clientKey) == 0 {
		return nil, fmt.Errorf("%s missing in secret", OpenStackClientKey)
	}

	var insecure bool
	if insecureStr, ok := data[OpenStackInsecure]; ok {
		switch string(insecureStr) {
		case "true":
			insecure = true
		case "false":
		default:
			return nil, fmt.Errorf("invalid value for boolean field %s: %s ", OpenStackInsecure, string(insecureStr))
		}
	}

	return &credentials{
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
	}, nil
}
