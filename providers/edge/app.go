package edge

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
)

func createApp(pod *v1.Pod) error {

	name := pod.Spec.Containers[0].Name

	running, err := isProcessRunning(name)
	if err != nil {
		return err
	}
	if running {
		err = stopProcess(name)
		if err != nil {
			return err
		}
	}
	exists, err := appExistsLocal(pathTooApplicationsDir, name)
	if err != nil {
		return err
	}
	if exists {
		err = removeApp(pathTooApplicationsDir, name, 3)
		if err != nil {
			return err
		}
	}
	err = downloadApp(name, "")
	if err != nil {
		return err
	}
	err = startApp(pod, pathTooApplicationsDir, name)
	if err != nil {
		return err
	}
	return nil
}

func deleteApp(name string) error {
	running, err := isProcessRunning(name)
	if err != nil {
		return err
	}
	if running {
		err = stopProcess(name)
		if err != nil {
			return err
		}
	}
	exists, err := appExistsLocal(pathTooApplicationsDir, name)
	if err != nil {
		return err
	}
	if exists {
		err = removeApp(pathTooApplicationsDir, name, 3)
		if err != nil {
			return err
		}
	}
	return nil
}

func appExistsLocal(pathTooApplicationsDir string, name string) (bool, error) {
	//fullAppPathExe := pathTooApplicationsDir + "/" + name + "/" + name + "exe"
	_, err := os.Stat(pathTooApplicationsDir + "/" + name)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func removeApp(pathTooApplicationsDir string, name string, retries int) error {
	appPath := pathTooApplicationsDir + "/" + name // + "/" + name + "exe"
	_, err := os.Stat(appPath)
	if err == nil {
		for i := 0; i < retries; i++ {
			log.Printf("Removing " + name)
			err = os.RemoveAll(appPath)
			if err == nil {
				return nil
			}
			time.Sleep(3 * time.Second)
		}
	}
	return err
}

func downloadApp(name string, version string) error {

	log.Printf("Downloading app %s version %s", name, version)
	resp, err := http.Get("http://localhost:8008" + "/application")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create("applications/scanner.zip")
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	Unzip("applications/scanner.zip", "applications"+"/"+name)
	return nil
}

func stopProcess(name string) error {
	log.Println("Stopping the " + name + " if running")
	procs, err := GetAllProcesses()
	if err != nil {
		return err
	}
	namedProcs := FindProcessByName(procs, name)

	for _, wp := range namedProcs {
		process, err := os.FindProcess(wp.ProcessID)
		if err != nil {
			return err
		}
		err = KillProcessAndChildren(process)
		if err != nil {
			return err
		}
	}
	return nil
}

func getProcessNameFromPodName(podname string) string {
	pos := strings.IndexAny(podname, "-")
	if pos != -1 {
		return podname[0:pos]
	}
	return ""
}

func startApp(pod *v1.Pod, pathTooApplicationsDir string, name string) error {
	log.Println("Starting " + name + " ...")
	bytes, err := ioutil.ReadFile(pathTooApplicationsDir + "/" + name + "/" + "args")
	if err != nil {
		return err
	}
	args := string(bytes)
	argsSlice := strings.Split(args, " ")

	//add any secrets passed as env variables
	containers := pod.Spec.Containers
	for _, c := range containers {
		envs := c.Env
		for _, e := range envs {
			key := e.Name
			val := e.Value
			argsSlice = append(argsSlice, "--"+key+"="+val)
		}
	}

	cmd := exec.Command(pathTooApplicationsDir+"/"+name+"/"+name+".exe", argsSlice...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	log.Println("Started " + name)
	return nil
}

func isProcessRunning(processName string) (bool, error) {
	procs, err := GetAllProcesses()
	if err != nil {
		return false, err
	}
	namedProcs := FindProcessByName(procs, processName)
	if len(namedProcs) > 0 {
		return true, nil
	}
	log.Println(processName + " not running")
	return false, nil
}

// Unzip unzip
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}
