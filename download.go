package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"runtime"

	"github.com/dustin/go-humanize"
	"code.google.com/p/go.crypto/ssh/terminal"
)

const API_URL = "http://api.ios.icj.me/firmwares.json"

type Firmware struct {
	Version  string `json:"version"`
	BuildID  string `json:"buildid"`
	URL      string `json:"url"`
	Date     string `json:"date"`
	Size     string `json:"size"`
	MD5      string `json:"md5sum"`
	SHA1     string `json:"sha1sum"`
	Filename string `json:"filename"`
}

type IndividualiTunes struct {
	Version         string `json:"version"`
	URL             string `json:"url"`
	SixtyFourBitURL string `json:"64biturl"`
	Date            string `json:"datefound"`
}

type Device struct {
	Name        string      `json:"name"`
	BoardConfig string      `json:"BoardConfig"`
	Platform    string      `json:"Platform"`
	CPID        string      `json:"cpid"`
	BDID        string      `json:"bdid"`
	Firmwares   []*Firmware `json:"firmwares"`
}

type APIJSON struct {
	Devices map[string]*Device             `json:"devices"`
	ITunes  map[string][]*IndividualiTunes `json:"itunes"`
}

func ProgressBar(progress int) (progressBar string) {

	var width int

	if runtime.GOOS == "windows" {
		// we'll just assume it's standard terminal width
		width = 80
	} else {
		width, _, _ = terminal.GetSize(0)
	}

	// take off 14 for extra info (e.g. percentage)
	width = width / 2 - 14

	// get the current progress
	currentProgress := (progress * width) / 100


	progressBar = "["

	for i := 0; i < currentProgress; i++ {
		progressBar = progressBar + "="
	}

	progressBar = progressBar + ">"

	for i := width; i > currentProgress; i-- {
		progressBar = progressBar + " "
	}

	progressBar = progressBar + "] " + fmt.Sprintf("%3d", progress) + "%%"

	return progressBar
}

func GetFirmwaresJSON() (parsed *APIJSON, err error) {
	response, err := http.Get(API_URL)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, &parsed)

	if err != nil {
		return nil, err
	}

	return parsed, err
}

// generate a SHA1 of the file, then compare it to a known one
func VerifyFile(filename string, sha1sum string) (result bool, err error) {
	path := filepath.Join(downloadDirectory, filename)
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer file.Close()

	h := sha1.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return false, err
	}

	bs := h.Sum(nil)

	return sha1sum == hex.EncodeToString(bs), nil
}

func DownloadIndividualFirmware(url string, filename string) (sha1sum string, err error) {

	//fmt.Println("Downloading to " + filepath.Join(downloadDirectory, filename) + "... ")

	downloadCount++

	out, err := os.Create(filepath.Join(downloadDirectory, filename))
	defer out.Close()

	h := sha1.New()
	mw := io.MultiWriter(out, h)

	if err != nil {
		return "", err
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	doneCh := make(chan struct{})
	go func() {
		size := resp.ContentLength
		downloaded := int64(0)
		buf := make([]byte, 128*1024)

		for {
			if n, _ := resp.Body.Read(buf); n > 0 {
				mw.Write(buf[:n])
				downloaded += int64(n)
				filesizeDownloaded += int64(n)
				pct := int((downloaded * 100) / size)

				fmt.Printf("\r(%d/%d) " + ProgressBar(pct) + " " + ProgressBar(int((filesizeDownloaded / totalFirmwareSize) * 100)), downloadCount, totalFirmwareCount)
			} else {
				break
			}
		}

		doneCh <- struct{}{}
	}()
	<-doneCh

	return hex.EncodeToString(h.Sum(nil)), err
}

// args!
var justCheck bool
var downloadDirectory string
var filesizeDownloaded int64
var totalFirmwareCount int
var totalFirmwareSize int64
var totalDeviceCount int
var downloadCount int

func init() {
	// parse the flags
	flag.BoolVar(&justCheck, "c", false, "just check the integrity of the currently downloaded files")
	flag.StringVar(&downloadDirectory, "d", "./", "the location to save/check IPSW files")

	flag.Parse()
}

func main() {


	// so we can catch interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	result, _ := GetFirmwaresJSON()

	go func() {
		for _ = range c {
			fmt.Println()

			fmt.Printf("Downloaded %v bytes\n", filesizeDownloaded)
			fmt.Printf("Exiting\n")
			os.Exit(0)
		}
	}()

	for _, info := range result.Devices {
		totalDeviceCount++
		for _, firmware := range info.Firmwares {
			if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); os.IsNotExist(err) {
				totalFirmwareCount++
				thisSize, _ := strconv.ParseUint(firmware.Size, 0, 0)
				totalFirmwareSize += int64(thisSize)
			}
		}
	}

	fmt.Printf("Downloading %v IPSW files for %v devices (%v)\n", totalFirmwareCount, totalDeviceCount, humanize.Bytes(uint64(totalFirmwareSize)))

	for identifier, deviceinfo := range result.Devices {

		fmt.Println("------------------")
		fmt.Println(identifier)
		fmt.Println("------------------")

		for _, firmware := range deviceinfo.Firmwares {
			fmt.Print("Checking if " + firmware.Filename + " exists... ")
			if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); os.IsNotExist(err) && !justCheck {

				fmt.Println("needs downloading ")
				shasum, err := DownloadIndividualFirmware(firmware.URL, firmware.Filename)

				if err != nil {
					fmt.Println(err)
				} else {

					// not sure if these will display properly
					if shasum == firmware.SHA1 {
						fmt.Println("✔")
					} else {
						fmt.Println("✘")
					}

					size, _ := strconv.ParseInt(firmware.Size, 0, 0)
					filesizeDownloaded += size
				}

			} else {
				fmt.Println("true")
				if justCheck {
					fmt.Print("\tfile is ok? ")
					fileOK, _ := VerifyFile(firmware.Filename, firmware.SHA1)
					fmt.Printf("%t\n", fileOK)
				}
			}
		}
	}

	fmt.Printf("Downloaded %v bytes\n", filesizeDownloaded)
}
