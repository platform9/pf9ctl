package auth

import (
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func tokenInfoToExecCredential(tokenInfo TokenInfo) *clientauthenticationv1beta1.ExecCredential {
	return &clientauthenticationv1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Kind:       "ExecCredential",
		},
		Status: &clientauthenticationv1beta1.ExecCredentialStatus{
			Token:               tokenInfo.Token,
			ExpirationTimestamp: &metav1.Time{Time: tokenInfo.ExpiresAt},
		},
	}
}

// TODO prettify output
func PrintTokenForKubectl(tokenInfo TokenInfo) error {
	ec := tokenInfoToExecCredential(tokenInfo)
	e := json.NewEncoder(os.Stdout)
	if err := e.Encode(ec); err != nil {
		return fmt.Errorf("could not write the ExecCredential: %w", err)
	}
	return nil
}
