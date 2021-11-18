package supportBundle

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
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
	fileloc     string
	err         error
	S3_Location string
	authfile    string
	msgfile     string
	lockfile    string
	hostOS      string

	//Errors returned from the functions
	ErrHostIP        = fmt.Errorf("Host IP not found")
	ErrRemove        = fmt.Errorf("Unable to remove bundle")
	ErrGenBundle     = fmt.Errorf("Unable to generate supportBundle in remote host")
	ErrUpload        = fmt.Errorf("Unable to upload supportBundle to S3")
	ErrPartialBundle = fmt.Errorf("Failed to generate complete supportBundle, generated partial bundle")

	//Timestamp used for generating targetfile
	Timestamp = time.Now()
)

func HostOS(exec cmdexec.Executor) {
	hostOS, err = pmk.ValidatePlatform(exec)
	if err != nil {
		zap.S().Fatalf("OS version is not supported")
	}

}

// To get the Host IP address
func HostIP(exec cmdexec.Executor) (string, error) {
	zap.S().Debug("Fetching HostIP")
	host, err := exec.RunWithStdout("bash", "-c", fmt.Sprintf("hostname -I"))
	if err != nil {
		zap.S().Error("Host IP Not found", err)
		return host, ErrHostIP
	}
	// If the host has multiple IPs
	host = strings.Split(host, " ")[0]
	return host, nil
}

// To upload pf9ctl log bundle to S3 bucket
func SupportBundleUpload(ctx objects.Config, allClients client.Client, isRemote bool) error {

	zap.S().Debugf("Received a call to upload pf9ctl supportBundle to %s bucket.\n", S3_BUCKET_NAME)
	HostOS(allClients.Executor)
	fileloc, err = GenSupportBundle(allClients.Executor, Timestamp, isRemote)
	if err != nil && err != ErrPartialBundle {
		if isRemote {
			zap.S().Debugf(color.Red("x ")+"Failed to generate supportBundle\n", err.Error())
			return err
		}
		zap.S().Debugf(color.Red("x ")+"Failed to generate supportBundle\n", err.Error())
	}

	// To get the HostIP
	hostIP, err := HostIP(allClients.Executor)
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
		ctx.MfaToken,
	)
	if err != nil {
		zap.S().Debug("Unable to locate keystone credentials: %s\n", err.Error())
		return fmt.Errorf("Unable to locate keystone credentials: %s\n", err.Error())
	}

	// To Fetch FQDN
	FQDN, err := keystone.FetchRegionFQDN(ctx.Fqdn, ctx.Region, auth)
	if err != nil {
		zap.S().Debug("unable to fetch fqdn: %w")
		return fmt.Errorf("unable to fetch fqdn: %w", err)
	}
	//To fetch FQDN from config if region given is invalid
	if FQDN == "" {
		FQDN = ctx.Fqdn
		FQDN = strings.Replace(FQDN, "https://", "", 1)
	}

	// S3 location to upload the file
	S3_Location = S3_Loc + "/" + FQDN + "/" + hostIP + "/"

	// To upload the pf9cli log bundle to S3 bucket
	errUpload := S3Upload(allClients.Executor)
	if errUpload != nil {
		zap.S().Debugf("Failed to upload pf9ctl supportBundle to %s bucket!! ", S3_BUCKET_NAME, errUpload)

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
	errremove := RemoveBundle(allClients.Executor)
	if errremove != nil {
		zap.S().Debug("Error removing generated bundle", errremove)
	}

	return nil
}

//To generate the targetfile name including the hostname and the timestamp
func GenTargetFilename(timestamp time.Time, hostname string) string {

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

//To upload supportBundle to the S3 location
func S3Upload(exec cmdexec.Executor) error {
	errUpload := exec.Run("bash", "-c", fmt.Sprintf("curl -T %s -H %s %s", fileloc,
		S3_ACL, S3_Location))
	if errUpload != nil {
		return ErrUpload
	}
	return nil
}

//To remove the supportBundle
func RemoveBundle(exec cmdexec.Executor) error {
	errremove := exec.Run("bash", "-c", fmt.Sprintf("rm -rf %s", fileloc))
	if errremove != nil {
		return ErrRemove
	}
	return nil
}

//This function is used to generate the support bundles.
//It copies all the log files specified into a directory and archives that given directory.
func GenSupportBundle(exec cmdexec.Executor, timestamp time.Time, isRemote bool) (string, error) {

	//Check whether the source directories exist in remote node.
	if !isRemote {
		_, errPf9 := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", util.Pf9LogDir))
		if err != nil {
			zap.S().Debug("Log files directory not Found!!", errPf9)
		}
	}

	_, errEtc := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", util.EtcDir))
	if errEtc != nil {
		zap.S().Debug("Log files directory not Found!! ", errEtc)
	}

	_, errVar := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", util.VarDir))
	if errVar != nil {
		zap.S().Debug("Log files directory not Found!! ", errVar)
	}

	_, errDmesg := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", util.DmesgLog))
	if errVar != nil {
		zap.S().Debug("Dmesg files not Found!! ", errDmesg)
	}

	//Assign specific files according to the platform
	//GenFiles(os, exec)

	if hostOS == "debian" {
		authfile = util.AuthDeb
		msgfile = util.MsgDeb
		lockfile = util.LockDeb
	} else {
		authfile = util.AuthRed
		msgfile = util.MsgRed
		lockfile = util.LockRed
	}

	_, errAuth := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", authfile))
	if errEtc != nil {
		zap.S().Debug("Auth files not Found!! ", errAuth)
	}

	_, errMsg := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", msgfile))
	if errEtc != nil {
		zap.S().Debug("Auth files not Found!! ", errMsg)
	}

	_, errLock := exec.RunWithStdout("bash", "-c", fmt.Sprintf("stat %s", lockfile))
	if errEtc != nil {
		zap.S().Debug("Auth files not Found!! ", errLock)
	}

	// To fetch the hostname of remote node
	hostname, err := exec.RunWithStdout("bash", "-c", fmt.Sprintf("hostname"))
	if err != nil {
		zap.S().Debug("Failed to fetch hostname ", err)
	}

	hostname = strings.TrimSpace(strings.Trim(hostname, "\n"))

	// To generate the targetfile name
	targetfile := GenTargetFilename(timestamp, hostname)

	if isRemote {
		// Generate supportBundle if any of Etc / var logs are present or both
		if errEtc == nil || errVar == nil {
			// Generation of supportBundle in remote host case.
			_, errbundle := exec.RunWithStdout("bash", "-c", fmt.Sprintf("tar -czf %s %s %s %s %s %s %s",
				targetfile, util.VarDir, util.EtcDir, util.DmesgLog, authfile, msgfile, lockfile))
			if errbundle != nil {
				zap.S().Debug("Failed to generate complete supportBundle, generated partial bundle (Remote Host)", errbundle)
			}

		} else {
			zap.S().Debug("Failed to generate supportBundle (Remote Host)", errVar, errEtc)
			return targetfile, ErrGenBundle
		}

		zap.S().Debug("Generated the pf9ctl supportBundle (Remote Host) successfully")
		return targetfile, nil

	} else {
		// Generation of supportBundle in local host case.
		_, errbundle := exec.RunWithStdout("bash", "-c", fmt.Sprintf("tar czf %s --directory=%s %s %s %s %s %s %s %s",
			targetfile, util.Pf9DirLoc, util.Pf9LogLoc, util.VarDir, util.EtcDir, util.DmesgLog, authfile, msgfile, lockfile))
		if errbundle != nil {
			zap.S().Debug("Failed to generate complete supportBundle, generated partial bundle", errbundle)
			return targetfile, ErrPartialBundle
		} else {
			zap.S().Debug("Generated the pf9ctl supportBundle successfully")
		}
		return targetfile, nil
	}

}

//To assign platform-specific auth,messages,dpkg/yum files
func GenFiles(os string, exec cmdexec.Executor) {

}
