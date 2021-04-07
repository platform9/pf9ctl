package supportBundle

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// These constants specifiy the S3 Bucket to upload supportBundle and its region
const (
	S3_BUCKET_NAME = "loguploads.platform9.com"
	S3_REGION      = "us-west-2"
	S3_ACL         = "x-amz-acl:bucket-owner-full-control"
	S3_Loc         = "https://s3-us-west-2.amazonaws.com/loguploads.platform9.com"
)

var (
	RemoteBundle bool = false
	hostname     string
	fileloc      string
	err          error
	S3_Location  string
)

// To get the Host IP address

func HostIP(allClients pmk.Client) (string, error) {

	zap.S().Debug("Fetching HostIP")

	host, err := allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("hostname -I"))
	if err != nil {
		zap.S().Error("Host IP Not found", err)
	}
	return host, err
}

// To upload pf9ctl log bundle to S3 bucket

func SupportBundleUpload(ctx pmk.Config, allClients pmk.Client) error {

	zap.S().Debugf("Received a call to upload pf9ctl supportBundle to %s bucket.\n", S3_BUCKET_NAME)

	timestamp := time.Now()

	// Generation of tar file of supportbundle (pf9ctl log files)
	if RemoteBundle {
		fileloc, err = genSupportBundleRemote(allClients, timestamp)
		if err != nil {
			zap.S().Debugf(color.Red("x ")+"Failed to generate supportBundle\n", err.Error())
		}
	} else {
		fileloc, err = genSupportBundle(allClients, timestamp)
		if err != nil {

			zap.S().Debugf(color.Red("x ")+"Failed to generate supportBundle\n", err.Error())
		}
	}

	// To get the HostIP
	hostIP, err := HostIP(allClients)
	if err != nil {
		zap.S().Debug("Unable to fetch Host IP")
	}

	//To remove extra spaces and lines after the IP
	hostIP = strings.TrimSpace(strings.Trim(hostIP, "\n"))

	// Fetch the keystone token.
	// This is used as a reference to the segment event.
	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)
	if err != nil {
		zap.S().Debug("Unable to locate keystone credentials: %s\n", err.Error())
		return fmt.Errorf("Unable to locate keystone credentials: %s\n", err.Error())
	}

	// To get the hostOS.
	hostOS, err := pmk.ValidatePlatform(allClients.Executor)
	if err != nil {
		zap.S().Debug("Error: Invalid host OS. " + err.Error())
		errStr := "Error: Invalid host OS. " + err.Error()
		return fmt.Errorf(errStr)
	}

	// To Fetch FQDN
	FQDN, err := pmk.FetchRegionFQDN(ctx, auth, hostOS)
	if err != nil {
		zap.S().Debug("unable to fetch fqdn: %w")
		return fmt.Errorf("unable to fetch fqdn: %w", err)
	}

	// S3 location to upload the file
	S3_Location = S3_Loc + "/" + FQDN + "/" + hostIP + "/"

	// To upload the pf9cli log bundle to S3 bucket

	err = allClients.Executor.Run("bash", "-c", fmt.Sprintf("curl -T %s -H %s %s", fileloc,
		S3_ACL, S3_Location))
	if err != nil {
		zap.S().Debugf("Failed to upload pf9ctl supportBundle to %s bucket!!", S3_BUCKET_NAME)

		if err := allClients.Segment.SendEvent("supportBundle upload Failed", auth, "Failed", ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for supportBundle. Error: %s", err.Error())
		}

	} else {
		zap.S().Debugf("Succesfully uploaded pf9ctl supportBundle to %s bucket at %s location \n",
			S3_BUCKET_NAME, S3_Location)
		if err := allClients.Segment.SendEvent("supportBundle upload Success", auth, "Success", ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for supportBundle. Error: %s", err.Error())
		}
	}

	// Remove the supportbundle after uploading to S3

	if RemoteBundle {
		err := allClients.Executor.Run("bash", "-c", fmt.Sprintf("rm -rf %s", fileloc))
		if err != nil {
			zap.S().Debug("Failed to remove supportbundle", err)
		}

	} else {
		err := os.Remove(fileloc)
		if err != nil {
			zap.S().Debug("Failed to remove supportbundle", err)
		}
	}
	return nil
}

/*This function is used to generate the support bundles.
It copies all the log files specified into a directory and archives that given directory. */
func genSupportBundle(allClients pmk.Client, timestamp time.Time) (string, error) {

	//Check whether the source directories exist.
	_, err := os.Stat(util.Pf9Dir)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Debug("Log files directory not Found")
		}
	}
	_, err = os.Stat(util.VarDir)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Debug("Log files directory not Found")
		}
	}
	_, err = os.Stat(util.EtcDir)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Debug("Log files directory not Found")
		}
	}

	// To fetch the hostname of given node
	hostname, err = os.Hostname()
	if err != nil {
		zap.S().Debug("Failed to fetch hostname", err)
	}

	// To generate the targetfile name
	targetfile := genTargetFilename(timestamp, hostname)

	// Generation of supportBundle in loacl host case.
	_, err = allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("tar czf %s --directory=%s pf9 %s %s",
		targetfile, util.Pf9DirLoc, util.VarDir, util.EtcDir))
	if err != nil {
		zap.S().Debug("Failed to generate supportBundle", err)
	}

	zap.S().Debug("Generated the pf9ctl supportBundle successfully")

	return targetfile, err
}

func genSupportBundleRemote(allClients pmk.Client, timestamp time.Time) (string, error) {

	//Check whether the source directories exist in remote node.
	_, err = allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("cd /etc | ls | grep -i pf9"))
	if err != nil {
		zap.S().Debug("Log files directory not Found!!")
	}

	_, err = allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("cd /var/log | ls | grep -i pf9"))
	if err != nil {
		zap.S().Debug("Log files directory not Found!!")
	}

	// To fetch the hostname of remote node
	hostname, err = allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("hostname"))
	if err != nil {
		zap.S().Debug("Failed to fetch hostname ", err)
	}
	hostname = strings.TrimSpace(strings.Trim(hostname, "\n"))

	// To generate the targetfile name
	targetfile := genTargetFilename(timestamp, hostname)

	// Generation of supportBundle in remote host case.
	_, err = allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("tar -czf %s %s %s",
		targetfile, util.VarDir, util.EtcDir))
	if err != nil {
		zap.S().Debug("Failed to generate supportBundle (Remote Host)", err)
	}

	zap.S().Debug("Generated the pf9ctl supportBundle (Remote Host) successfully")

	return targetfile, err
}

//To generate the targetfile name including the hostname and the timestamp
func genTargetFilename(timestamp time.Time, hostname string) string {

	//timestamp format for the archive file(Note:UTC Time is taken)
	//File Format - hostname-yy-mm-dd-hours-minutes-seconds.tar.gz
	//Sample File Format- test-dev-vm-2021-04-01-16-29-17.tar.gz
	hour := strconv.Itoa(timestamp.Hour())
	minutes := strconv.Itoa(timestamp.Minute())
	seconds := strconv.Itoa(timestamp.Second())
	layout := timestamp.Format("2006-01-02")
	tarname := hostname + "-" + layout + "-" + hour + "-" + minutes + "-" + seconds
	tarzipname := tarname + ".tar.gz"
	targetfile := "/tmp/" + tarzipname
	return targetfile
}
