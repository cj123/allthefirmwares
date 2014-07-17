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
	b, err := ioutil.ReadFile(filepath.Join(downloadDirectory, filename))

	h := sha1.New()

	h.Write(b)

	bs := h.Sum(nil)

	return sha1sum == hex.EncodeToString(bs), err
}

func DownloadIndividualFirmware(url string, filename string) (err error) {

	fmt.Print("Downloading " + filename + " to " + filepath.Join(downloadDirectory, filename) + "... ")

	out, err := os.Create(filepath.Join(downloadDirectory, filename))
	defer out.Close()

	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)

	fmt.Println("Done!")

	return err
}

// args!
var justCheck bool
var noCheck bool
var downloadDirectory string

func init() {
	// parse the flags
	flag.BoolVar(&justCheck, "c", false, "just check the integrity of the currently downloaded files")
	flag.BoolVar(&noCheck, "z", false, "don't check files after they have been downloaded (faster)")
	flag.StringVar(&downloadDirectory, "d", "./", "the location to save/check IPSW files")
	flag.Parse()
}

func main() {
	// so we can catch interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	result, _ := GetFirmwaresJSON()

	// total size downloaded
	var filesizeDownloaded int64 = 0

	go func() {
		for _ = range c {
			fmt.Println()

			fmt.Printf("Downloaded %v bytes\n", filesizeDownloaded)
			fmt.Printf("Ending")
			os.Exit(0)
		}
	}()

	for identifier, deviceinfo := range result.Devices {

		fmt.Println("------------------")
		fmt.Println(identifier)
		fmt.Println("------------------")

		for _, firmware := range deviceinfo.Firmwares {
			fmt.Print("Checking if " + firmware.Filename + " exists... ")
			if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); os.IsNotExist(err) && !justCheck {

				fmt.Println("needs downloading ")
				err = DownloadIndividualFirmware(firmware.URL, firmware.Filename)

				if err != nil {
					fmt.Println(err)
				} else {
					if !noCheck {
						fileOK, _ := VerifyFile(firmware.Filename, firmware.SHA1)

						fmt.Printf("file is ok? %t\n", fileOK)
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
