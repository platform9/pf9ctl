package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//This function takes a source string and destination string as parameters.
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must not exist.
// Symlinks are ignored and skipped.
func CopyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

//This function will take the absolute path as an input and convert it into the relative path
func expand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, path[1:]), nil
}

//This fucntion recursively archives all the files present in the directory given.
//Parameters: 1. Source Directory
//            2. Destination Directory
func RecursiveZip(pathToZip, destinationPath string) error {
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	myZip := zip.NewWriter(destinationFile)
	err = filepath.Walk(pathToZip, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(filePath, pathToZip)
		zipFile, err := myZip.Create(relPath)
		if err != nil {
			return err
		}

		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, fsFile)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = myZip.Close()
	if err != nil {
		return err
	}
	return nil
}

//This function is used to generate the support bundles. It copies all the log files specified into a directory and archives that given directory
func gensupportbundle() {
	t := time.Now()

	file := "~/pf9/log/"
	source, err := expand(file)
	fmt.Println(source, err)
	dest := "~/pf9/copy/pf9/log/"
	target, err := expand(dest)
	fmt.Println(target, err)
	CopyDir(source, target)

	file1 := "~/pf9/db/"
	source1, err := expand(file1)
	fmt.Println(source1, err)
	dest1 := "~/pf9/copy/pf9/db/"
	target1, err := expand(dest1)
	fmt.Println(target1, err)
	CopyDir(source1, target1)

	file2 := "/var/log/pf9/"
	source2, err := expand(file2)
	fmt.Println(source2, err)
	dest2 := "~/pf9/copy/var/log/pf9/"
	target2, err := expand(dest2)
	fmt.Println(target2, err)
	CopyDir(source2, target2)

	file3 := "/etc/pf9/"
	source3, err := expand(file3)
	fmt.Println(source3, err)
	dest3 := "~/pf9/copy/etc/pf9/"
	target3, err := expand(dest3)
	fmt.Println(target3, err)
	CopyDir(source3, target3)

	//Storing the hostname for the given node
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	//timestamp format for the zip file
	layout := t.Format("2006-01-02")
	h := t.Hour()
	s1 := strconv.Itoa(h)
	m := t.Minute()
	s2 := strconv.Itoa(m)
	s := t.Second()
	s3 := strconv.Itoa(s)

	destination := "/tmp/" + name + "-" + layout + "-" + s1 + "-" + s2 + "-" + s3 + ".tgz"
	targetfile, err := expand(destination)
	fmt.Println(targetfile, err)

	//This function will archive the source directory,subdirectories,files and place the archived file in the targetfile directory
	copydirectory := "~/pf9/copy/"
	copy1, err := expand(copydirectory)
	RecursiveZip(copy1, targetfile)
	fmt.Println("Zipped Successfully")

	//This function will remove all the contents of the directory
	err1 := os.RemoveAll(copy1)
	if err1 != nil {
		log.Fatal(err1)
	}

}
func main() {
	gensupportbundle()
}
