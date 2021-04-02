package supportBundle

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver/v3"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/plus3it/gorecurcopy"
	"go.uber.org/zap"
)

// These contants specifiy the S3 Bucket to upload supportBundle and its region
const (
	S3_BUCKET_NAME = "loguploads.platform9.com"
	S3_REGION      = "us-west-2"
	S3_ACL         = "x-amz-acl:bucket-owner-full-control"
	S3_Loc         = "https://s3-us-west-2.amazonaws.com/loguploads.platform9.com"
)

// To get the Host IP address

func HostIP(allClients pmk.Client) (string, error) {

	zap.S().Debug("Fetching HostIP")

	host, err := allClients.Executor.RunWithStdout("bash", "-c", fmt.Sprintf("hostname -I"))
	if err != nil {
		zap.S().Error("Host IP Not found", err)
	}
	return host, nil
}

// To upload pf9ctl log bundle to S3 bucket

func SupportBundleUpload(ctx pmk.Config, allClients pmk.Client) error {

	zap.S().Debugf("Received a call to upload pf9ctl log bundle to %s bucket.\n", S3_BUCKET_NAME)

	// Generation of tar file of supportbundle (pf9ctl log files)
	fileloc, err := Gensupportbundle()
	if err != nil {
		fmt.Printf(color.Green("x ") + "Failed to generate pf9ctl log bundle\n")
		zap.S().Debug("Failed to generate pf9ctl log bundle\n")
	}

	// To get the HostIP
	host, _ := HostIP(allClients)
	//To remove extra spaces and lines after the IP
	host = strings.TrimSpace(strings.Trim(host, "\n"))

	// Fetch the keystone token.
	// This is used as a reference to the segment event.
	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s\n", err.Error())
	}

	// To get the hostOS.
	hostOS, err := pmk.ValidatePlatform(allClients.Executor)
	if err != nil {
		errStr := "Error: Invalid host OS. " + err.Error()
		return fmt.Errorf(errStr)
	}

	// To Fetch Fqnd
	fqdn, err := pmk.FetchInstallerURL(ctx, auth, hostOS)
	if err != nil {
		return fmt.Errorf("unable to fetch fqdn: %w", err)
	}

	// S3 location to upload the file
	S3_Location := S3_Loc + "/" + fqdn + "/" + host + "/"

	// To upload the pf9cli log bundle to S3 bucket
	err = allClients.Executor.Run("bash", "-c", fmt.Sprintf("curl -T %s -H %s %s", fileloc, S3_ACL, S3_Location))
	if err != nil {
		zap.S().Error("Failed to upload pf9ctl log bundle to %s bucket!!", S3_BUCKET_NAME, err)
	}

	//To remove the supportbundle after getting uploaded
	err = os.Remove(fileloc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(color.Green("✓ ")+"Succesfully uploaded pf9ctl log bundle to %s bucket\n", S3_BUCKET_NAME)
	zap.S().Debugf("Succesfully uploaded pf9ctl log bundle to %s bucket\n", S3_BUCKET_NAME)

	return nil
}

//This function is used to generate the support bundles. It copies all the log files specified into a directory and archives that given directory
func Gensupportbundle() (string, error) {

	//Checking whether the source directories exist
	_, err := os.Stat(util.Pf9Dir)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Error("Directory not Found", err)
		}
	}
	_, err = os.Stat(util.DirVar)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Error("Directory not Found", err)
		}
	}
	_, err = os.Stat(util.DirEtc)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Error("Directory not Found", err)
		}
	}

	//Recursively copying the contents of source directory to destination directory
	//Function:gorecurcopy.CopyDirectory(Source Directory,Destination Directory)
	err = gorecurcopy.CopyDirectory(util.Pf9Dir, util.DestDirPf9)
	if err != nil {
		zap.S().Error("Error in copying directory ", err)
	}
	err = gorecurcopy.CopyDirectory(util.DirVar, util.DestDirvar)
	if err != nil {
		zap.S().Error("Error in copying  directory ", err)
	}
	err = gorecurcopy.CopyDirectory(util.DirEtc, util.DestDirPf9EtcDir)
	if err != nil {
		zap.S().Error("Error in copying  directory ", err)
	}
	//Storing the hostname for the given node
	hostname, err := os.Hostname()
	if err != nil {
		zap.S().Error("Error fetching hostname", err)
	}

	//timestamp format for the zip file
	timestamp := time.Now()
	tarname := hostname + "-" + timestamp.Format("2006-01-02") + "-" + strconv.Itoa(timestamp.Hour()) + "-" + strconv.Itoa(timestamp.Minute()) + "-" + strconv.Itoa(timestamp.Second())
	tarzipname := tarname + ".tar.gz"
	targetfile := "/tmp/" + tarzipname
	destDir := "/tmp/" + tarname

	//Renaming the copied directory according to the format
	os.Rename(util.DestDir, destDir)

	//This function archives the contents of the source directory and places it in the archive file
	//Function:archiver.Archive(Source Directory,Archive file)
	err = archiver.Archive([]string{destDir}, targetfile)
	if err != nil {
		zap.S().Error("Error in creating the archive of pf9ctl log files", err)
	}
	zap.S().Debug("Zipped the pf9ctl log files successfully")

	//This function will remove all the contents of the copied directory
	err = os.RemoveAll(destDir)
	if err != nil {
		log.Fatal(err)
	}
	return targetfile, nil
}
