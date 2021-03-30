package supportBundle

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mholt/archiver/v3"
	"github.com/plus3it/gorecurcopy"

	//"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

/*
// These contants specifiy the S3 Bucket to upload supportBundle and its region
const (
	S3_BUCKET_NAME = "loguploads.platform9.com"
	S3_REGION      = "us-west-2"
)
*/
/*
// To initialise the new s3 session with in the specified region.
func init() {
	s3session = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String(S3_REGION),
	})))
}

//This will upload the file(object) to s3 bucket.
func uploadFileToS3(filename string) (resp *s3.PutObjectOutput) {
	//This is to check the zip file exists.
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}


	zap.S().Infof("Uploading files %s", f)
	resp, err = s3session.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: aws.String(S3_BUCKET_NAME),
		Key:    aws.String(filename),
	})
	if err != nil {
		zap.S().Fatalf("Uploading supportBundle Falied")
	}

return resp*/

//This will upload the file(object) to s3 bucket.
/*
func uploadFileToS3(filename string) error {
	//This is to check the zip file exists.
	file, err := os.Open(filename)
	if err != nil {
		zap.S().Errorf("Unable to open file %q, %v", err)
	}

	defer file.Close()

	// To initialise the new s3 session with in the specified region.
	s3session, err := session.NewSession(&aws.Config{
		Region: aws.String(S3_REGION)},
	)
	if err != nil {
		zap.S().Errorf("Unable to create S3 Session")
	}

	uploader := s3manager.NewUploader(s3session)

	zap.S().Infof("Uploading files %s", filename)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(S3_BUCKET_NAME),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		// Print the error and exit.
		zap.S().Errorf("Unable to upload %q to %q, %v", filename, S3_BUCKET_NAME, err)
	}
	return nil
}

func SupportBundle() {
	tarname, err := Gensupportbundle()
	if err != nil {
		zap.S().Info("Support Bundle not generated")
	}

	fmt.Println(tarname)

	folder := "/tmp/support/"
	files, _ := ioutil.ReadDir(folder)
	fmt.Println(files)
	for _, file := range files {
		if file.Name() == "hello.txt" {
			_ = uploadFileToS3(file.Name())
		}
	}
	fmt.Println("Uploaded successfully")

	/*err1 := os.RemoveAll(folder)
	if err1 != nil {
		log.Fatal(err1)
	}

}
*/

//This function is used to generate the support bundles. It copies all the log files specified into a directory and archives that given directory
func Gensupportbundle() {
	//Checking whether the source directories exist
	_, e := os.Stat(util.Pf9Dir)
	if e != nil {

		if os.IsNotExist(e) {
			zap.S().Debug("Directory ~/pf9 not Found !!")
		}
	}

	_, e1 := os.Stat(util.DirVar)
	if e1 != nil {

		if os.IsNotExist(e1) {
			zap.S().Debug("Directory /var/pf9/log not Found !!")
		}
	}
	_, e2 := os.Stat(util.DirEtc)
	if e2 != nil {

		if os.IsNotExist(e1) {
			zap.S().Debug("Directory /etc/pf9 not Found !!")
		}
	}

	//Recursively copying the contents of source directory to destination directory
	//Function:gorecurcopy.CopyDirectory(Source Directory,Destination Directory)
	err1 := gorecurcopy.CopyDirectory(util.Pf9Dir, util.DestDirPf9)
	if err1 != nil {
		zap.S().Debug("Error in copying ~/pf9 directory ")
	}
	err2 := gorecurcopy.CopyDirectory(util.DirVar, util.DestDirvar)
	if err2 != nil {
		zap.S().Debug("Error in copying /var/pf9/log directory ")
	}
	err3 := gorecurcopy.CopyDirectory(util.DirEtc, util.DestDirPf9EtcDir)
	if err3 != nil {
		zap.S().Debug("Error in copying /etc/pf9 directory ")
	}

	//Storing the hostname for the given node
	name, err := os.Hostname()
	if err != nil {
		zap.S().Debug("Error fetching hostname")
	}

	t := time.Now()
	//timestamp format for the zip file
	layout := t.Format("2006-01-02")
	h := t.Hour()
	s1 := strconv.Itoa(h)
	m := t.Minute()
	s2 := strconv.Itoa(m)
	s := t.Second()
	s3 := strconv.Itoa(s)

	tarname := name + "-" + layout + "-" + s1 + "-" + s2 + "-" + s3
	targetfile := "/tmp/support/" + tarname + ".tar.gz"
	DestDir := util.DestDir + "-" + tarname
	//Renaming the copied directory according to the format
	os.Rename(util.DestDir, DestDir)

	//This function archives the contents of the source directory and places it in the archive file
	//Function:archiver.Archive(Source Directory,Archive file)
	err5 := archiver.Archive([]string{DestDir}, targetfile)
	if err5 != nil {
		zap.S().Debug("Error in creating the archive file")
	}
	zap.S().Debug("Zipped Successfully")

	//This function will remove all the contents of the copied directory
	err6 := os.RemoveAll(DestDir)
	if err6 != nil {
		log.Fatal(err6)
	}
	//return tarname, nil
}
