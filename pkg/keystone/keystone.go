// Copyright © 2020 The Platform9 Systems Inc.

package keystone

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"go.uber.org/zap"
)

type KeystoneAuth struct {
	DUFqdn    string
	Token     string
	UserID    string
	ProjectID string
	Email     string
}

type Keystone interface {
	GetAuth(username, password, tenant string, mfa string) (KeystoneAuth, error)
}

type KeystoneImpl struct {
	fqdn string
}

func NewKeystone(fqdn string) Keystone {
	return KeystoneImpl{fqdn}
}

func (k KeystoneImpl) GetAuth(
	username,
	password,
	tenant string,
	mfa string) (auth KeystoneAuth, err error) {

	zap.S().Debugf("Received a call to fetch keystone authentication for fqdn: %s and user: %s and tenant: %s, mfa_token: %s\n", k.fqdn, username, tenant, mfa)

	url := fmt.Sprintf("%s/keystone", k.fqdn)

	clientOpts := gophercloud.AuthOptions{
		IdentityEndpoint: url,
		Username:         username,
		Password:         password,
		Passcode:         mfa,
		TenantName:       tenant,
		DomainName:       "default",
	}

	tokenOpts := tokens.AuthOptions{
		Username:   username,
		Password:   password,
		Passcode:   mfa,
		DomainName: "default",
		Scope: tokens.Scope{
			ProjectName: tenant,
			DomainName:  "default",
		},
	}

	// try with tenant as ProjectName
	auth, err = requestKeystone(clientOpts, tokenOpts)
	if err != nil {
		// try with tenant as ProjectID
		clientOpts.TenantID = tenant
		clientOpts.TenantName = ""
		tokenOpts.Scope.ProjectID = tenant
		tokenOpts.Scope.ProjectName = ""
		tokenOpts.Scope.DomainName = ""
		auth, err = requestKeystone(clientOpts, tokenOpts)
		if err != nil {
			return auth, err
		}

func requestKeystone(clientOpts gophercloud.AuthOptions, tokenOpts tokens.AuthOptions) (auth KeystoneAuth, err error) {
	provider, err := openstack.AuthenticatedClient(clientOpts)
	if err != nil {
		return auth, err
	}

	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return auth, err
	}

	result := tokens.Create(client, &tokenOpts)

	token, err := result.ExtractTokenID()
	if err != nil {
		return auth, err
	}

	user, err := result.ExtractUser()
	if err != nil {
		return auth, err
	}

	project, err := result.ExtractProject()
	if err != nil {
		return auth, err
	}

	return KeystoneAuth{
		Token:     token,
		UserID:    user.ID,
		ProjectID: project.ID,
		Email:     user.Name,
	}, nil
}
