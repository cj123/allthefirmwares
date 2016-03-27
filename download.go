package main

import (
	"bytes"
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
	"runtime"
	"text/template"
	_ "crypto/sha512"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/dustin/go-humanize"
)

const API_URL = "https://api.ipsw.me/v2.1/firmwares.json"

type Device struct {
	Name        string      `json:"name"`
	BoardConfig string      `json:"BoardConfig"`
	Platform    string      `json:"platform"`
	CPID        int         `json:"cpid"`
	BDID        int         `json:"bdid"`
	Firmwares   []*Firmware `json:"firmwares"`
}

type Firmware struct {
	Identifier  string `json:"identifier,omitempty"`
	Version     string `json:"version"`
	Device      string `json:"device,omitempty"`
	BuildID     string `json:"buildid"`
	SHA1        string `json:"sha1sum"`
	MD5         string `json:"md5sum"`
	Size        int64  `json:"size"`
	ReleaseDate string `json:"releasedate,omitempty"`
	UploadDate  string `json:"uploaddate"`
	URL         string `json:"url"`
	Signed      bool   `json:"signed"`
	Filename    string `json:"filename"`
}

type iTunes struct {
	Platform        string `json:"platform,omitempty"`
	Version         string `json:"version"`
	UploadDate      string `json:"uploaddate"`
	URL             string `json:"url"`
	SixtyFourBitURL string `json:"64biturl,omitempty"`
	ReleaseDate     string `json:"releasedate,omitempty"`
}
type APIJSON struct {
	Devices map[string]*Device   `json:"devices"`
	ITunes  map[string][]*iTunes `json:"iTunes"`
}

// args!
var justCheck, redownloadIfBroken, downloadSigned bool
var downloadDirectory, downloadDirectoryTempl, currentIPSW, onlyDevice string
var filesizeDownloaded, totalFirmwareSize int64
var totalFirmwareCount, totalDeviceCount, downloadCount int

func init() {
	// parse the flags
	flag.BoolVar(&justCheck, "c", false, "just check the integrity of the currently downloaded files")
	flag.BoolVar(&redownloadIfBroken, "r", false, "redownload the file if it fails verification (w/ -c)")
	flag.BoolVar(&downloadSigned, "s", false, "only download signed firmwares")
	flag.StringVar(&downloadDirectoryTempl, "d", "./", "the location to save/check IPSW files.\n\t Can include templates e.g. {{.Identifier}} or {{.BuildID}}")
	flag.StringVar(&onlyDevice, "i", "", "only download for the specified device")
	flag.Parse()
}

// returns a progress bar fitting the terminal width given a progress percentage
func ProgressBar(progress int) (progressBar string) {

	var width int

	if runtime.GOOS == "windows" {
		// we'll just assume it's standard terminal width
		width = 80
	} else {
		width, _, _ = terminal.GetSize(0)
	}

	// take off 26 for extra info (e.g. percentage)
	width = width - 26

	// get the current progress
	currentProgress := (progress * width) / 100

	progressBar = "["

	// fill up progress
	for i := 0; i < currentProgress; i++ {
		progressBar = progressBar + "="
	}

	progressBar = progressBar + ">"

	// fill the rest with spaces
	for i := width; i > currentProgress; i-- {
		progressBar = progressBar + " "
	}

	// end the progressbar
	progressBar = progressBar + "] " + fmt.Sprintf("%3d", progress) + "%%"

	return progressBar
}

// get the JSON from API_URL and parse it
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
func (fw *Firmware) VerifyFile() (result bool, err error) {
	path := filepath.Join(downloadDirectory, fw.Filename)
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

	return fw.SHA1 == hex.EncodeToString(bs), nil
}

// Download the given firmware
func (fw *Firmware) Download() (sha1sum string, err error) {

	//fmt.Println("Downloading to " + filepath.Join(downloadDirectory, filename) + "... ")

	downloadCount++

	out, err := os.Create(filepath.Join(downloadDirectory, fw.Filename))
	defer out.Close()

	h := sha1.New()
	mw := io.MultiWriter(out, h)

	if err != nil {
		return "", err
	}

	resp, err := http.Get(fw.URL)
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

				fmt.Printf("\r(%d/%d) "+ProgressBar(pct)+" of %4v", downloadCount, totalFirmwareCount, humanize.Bytes(uint64(size)))
			} else {
				break
			}
		}

		doneCh <- struct{}{}
	}()
	<-doneCh

	return hex.EncodeToString(h.Sum(nil)), err
}

// Read the directory to download from the args, parsing templates if they exist
func (fw *Firmware) ParseDownloadDir(identifier string, makeDir bool) {

	directoryString := new(bytes.Buffer)

	tmpl, err := template.New("firmware").Parse(downloadDirectoryTempl)

	// add the identifier, for simplicity
	fw.Identifier = identifier

	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(directoryString, fw)

	if err != nil {
		panic(err)
	}

	downloadDirectory = directoryString.String()

	// make the directory
	if makeDir {
		os.MkdirAll(downloadDirectory, 0700)
	}
}

func main() {

	// so we can catch interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			fmt.Println()

			fmt.Printf("Downloaded %v\n", humanize.Bytes(uint64(filesizeDownloaded)))

			os.Exit(0)
		}
	}()

	result, err := GetFirmwaresJSON()

	if err != nil {
		panic(err)
	}

	if !justCheck {

		// calculate the size of the download

		for identifier, info := range result.Devices {
			if onlyDevice != "" && identifier != onlyDevice {
				continue
			}

			totalDeviceCount++
			for _, firmware := range info.Firmwares {

				// don't download unsigned firmwares if we just want signed ones
				if downloadSigned && !firmware.Signed {
					continue
				}

				firmware.ParseDownloadDir(identifier, false)

				if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); os.IsNotExist(err) {
					totalFirmwareCount++
					totalFirmwareSize += firmware.Size
				}
			}
		}

		fmt.Printf("Downloading %v IPSW files for %v device(s) (%v)\n", totalFirmwareCount, totalDeviceCount, humanize.Bytes(uint64(totalFirmwareSize)))
	}

	for identifier, deviceinfo := range result.Devices {

		if onlyDevice != "" && identifier != onlyDevice {
			continue
		}

		fmt.Printf("\nDevice: %s (%s) - %v firmwares\n", deviceinfo.Name, identifier, len(deviceinfo.Firmwares))
		fmt.Println("------------------------------------------------------\n")

		for _, firmware := range deviceinfo.Firmwares {

			firmware.ParseDownloadDir(identifier, !justCheck)

			// don't download unsigned firmwares if we just want signed ones
			if downloadSigned && !firmware.Signed {
				continue
			}

			fmt.Print("Checking if " + firmware.Filename + " exists... ")
			if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); os.IsNotExist(err) && !justCheck {

				fmt.Println("needs downloading")
				shasum, err := firmware.Download()

				if err != nil {
					fmt.Println(err)
				} else {

					// not sure if these will display properly
					if shasum == firmware.SHA1 {
						fmt.Println("✔")
					} else {
						fmt.Println("✘")
					}
				}

			} else if _, err := os.Stat(filepath.Join(downloadDirectory, firmware.Filename)); !os.IsNotExist(err) && justCheck {
				fmt.Println("true")

				fmt.Print("\tchecking file... ")

				if fileOK, _ := firmware.VerifyFile(); fileOK {
					fmt.Println("✔ ok")
				} else {
					fmt.Println("✘ bad")
					if redownloadIfBroken {
						fmt.Println("Redownloading...")
						shasum, err := firmware.Download()

						if err != nil {
							fmt.Println(err)
						}

						if shasum == firmware.SHA1 {
							fmt.Println("✔")
						} else {
							fmt.Println("✘")
						}
					}
				}

			} else {
				fmt.Println("false")
			}
		}
	}

	fmt.Printf("Downloaded %v\n", humanize.Bytes(uint64(filesizeDownloaded)))
}
