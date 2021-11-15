package keystone

import (
	"fmt"

	"go.uber.org/zap"
)

func FetchRegionFQDN(fqdn string, region string, auth KeystoneAuth) (string, error) {

	// "regionInfo" service will have endpoint information. So fetch it's service ID.
	regionInfoServiceID, err := GetServiceID(fqdn, auth, "regionInfo")
	if err != nil {
		return "", fmt.Errorf("Failed to fetch installer URL, Error: %s", err)
	}
	zap.S().Debug("Service ID fetched : ", regionInfoServiceID)

	// Fetch the endpoint based on region name.
	endpointURL, err := GetEndpointForRegion(fqdn, auth, region, regionInfoServiceID)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch installer URL, Error: %s", err)
	}
	zap.S().Debug("endpointURL fetched : ", endpointURL)
	return endpointURL, nil
}
