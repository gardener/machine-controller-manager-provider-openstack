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

package cloudprovider

const (
	// OpenStackAuthURL is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackAuthURL string = "authURL"
	// OpenStackCACert is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackCACert string = "caCert"
	// OpenStackInsecure is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackInsecure string = "insecure"
	// OpenStackDomainName is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackDomainName string = "domainName"
	// OpenStackDomainID is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackDomainID string = "domainID"
	// OpenStackTenantName is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackTenantName string = "tenantName"
	// OpenStackTenantID is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackTenantID string = "tenantID"
	// OpenStackUserDomainName is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackUserDomainName string = "userDomainName"
	// OpenStackUserDomainID is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackUserDomainID string = "userDomainID"
	// OpenStackUsername is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackUsername string = "username"
	// OpenStackPassword is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackPassword string = "password"
	// OpenStackClientCert is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackClientCert string = "clientCert"
	// OpenStackClientKey is a constant for a key name that is part of the OpenStack cloud Credentials.
	OpenStackClientKey string = "clientKey"

	// UserData is a constant for a key name whose value contains data passed to the server e.g. CloudInit scripts.
	UserData string = "userData"

	// ServerTagClusterPrefix is the prefix used for tags denoting the cluster this server belongs to.
	ServerTagClusterPrefix = "kubernetes.io-cluster-"
	// ServerTagRolePrefix is the prefix used for tags denoting the role of the server.
	ServerTagRolePrefix    = "kubernetes.io-role-"
)
