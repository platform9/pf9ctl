package resmgr
import (
	"testing"
	rhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/platform9/pf9ctl/pkg/util"
	"fmt"
)


func TestRetryHTTP(t *testing.T) {

	client := rhttp.NewClient()
	
	client.Logger = &util.ZapWrapper{}
	
    req, err := rhttp.NewRequest("GET", "http://www.google.com", nil)
	if err != nil {
		t.Errorf("Unable to create a new request: %s", err)
	}
	fmt.Printf("Send the request now")
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to send request to the client: %s", err)
	}
	defer resp.Body.Close()

}