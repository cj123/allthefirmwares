package main

import (
	"bytes"
	"crypto/sha1"
	_ "crypto/sha512"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"text/template"

	"github.com/cheggaaa/pb"
	"github.com/cj123/go-ipsw/api"
	"github.com/dustin/go-humanize"
)

var (
	ipswClient = api.NewIPSWClient("https://api.ipsw.me/v4", nil)

	// flags
	verifyIntegrity, reDownloadOnVerificationFailed, downloadSigned bool
	downloadDirectoryTemplate, specifiedDevice                      string

	// counters
	downloadedSize, totalFirmwareSize    uint64
	totalFirmwareCount, totalDeviceCount int
)

func init() {
	flag.BoolVar(&verifyIntegrity, "c", false, "just check the integrity of the currently downloaded files (if any)")
	flag.BoolVar(&reDownloadOnVerificationFailed, "r", false, "redownload the file if it fails verification (w/ -c)")
	flag.BoolVar(&downloadSigned, "s", false, "only download signed firmwares")
	flag.StringVar(&downloadDirectoryTemplate, "d", "./", "the location to save/check IPSW files.\n\tCan include templates e.g. {{.Identifier}} or {{.BuildID}}")
	flag.StringVar(&specifiedDevice, "i", "", "only download for the specified device")
	flag.Parse()
}

func main() {
	// catch interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			// sig is a ^C, handle it
			fmt.Println()
			log.Printf("Downloaded %v\n", humanize.Bytes(uint64(downloadedSize)))

			os.Exit(0)
		}
	}()

	log.Printf("Gathering IPSW information...")

	devices, err := ipswClient.Devices(false)

	if err != nil {
		log.Fatalf("Unable to retrieve firmware information, err: %s", err)
	}

	firmwaresToDownload := make(map[api.BaseDevice][]api.Firmware)

	for _, device := range devices {
		if specifiedDevice != "" && device.Identifier != specifiedDevice {
			continue
		}

		deviceInformation, err := ipswClient.DeviceInformation(device.Identifier)

		if err != nil {
			log.Printf("Could not get firmwares for device: %s, err: %s", device.Identifier, err)
		}

		totalDeviceCount++

		for _, ipsw := range deviceInformation.Firmwares {
			if downloadSigned && !ipsw.Signed {
				continue
			}

			directory, err := parseDownloadDirectory(&ipsw, device.Identifier)

			if err != nil {
				log.Printf("Unable to parse download directory, err: %s", err)
				continue
			}

			downloadPath := filepath.Join(directory, filepath.Base(ipsw.URL))

			if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
				totalFirmwareCount++
				totalFirmwareSize += ipsw.Filesize

				if firmwaresToDownload[device] == nil {
					firmwaresToDownload[device] = make([]api.Firmware, 0)
				}

				firmwaresToDownload[device] = append(firmwaresToDownload[device], ipsw)
			}
		}
	}

	if !verifyIntegrity {
		log.Printf("Downloading: %v IPSW files for %v device(s) (%v)", totalFirmwareCount, totalDeviceCount, humanize.Bytes(totalFirmwareSize))
	}

	for device, firmwares := range firmwaresToDownload {
		if !verifyIntegrity {
			log.Printf("Downloading %d firmwares for %s", len(firmwares), device.Name)
		}

		for _, ipsw := range firmwares {
			if downloadSigned && !ipsw.Signed {
				continue
			}

			filename := filepath.Base(ipsw.URL)

			directory, err := parseDownloadDirectory(&ipsw, device.Identifier)

			if err != nil {
				log.Printf("Unable to parse download directory, err: %s", err)
				continue
			}

			// ensure download directory exists
			if !verifyIntegrity {
				err := os.MkdirAll(directory, 0700)

				if err != nil {
					log.Printf("Unable to create download directory: %s, err: %s", directory, err)
					break
				}
			}

			downloadPath := filepath.Join(directory, filename)

			_, err = os.Stat(downloadPath)

			if os.IsNotExist(err) && !verifyIntegrity {
				for {
					err := downloadWithProgressBar(&ipsw, downloadPath)

					if err == nil || !reDownloadOnVerificationFailed {
						break
					}
				}
			} else if err == nil && verifyIntegrity {
				fileOK, err := verify(downloadPath, ipsw.SHA1Sum)

				if err != nil {
					log.Printf("Error verifying: %s, err: %s", filename, err)
				}

				if fileOK {
					log.Printf("%s verified successfully", filename)
					continue
				}

				log.Printf("%s did not verify successfully", filename)

				if reDownloadOnVerificationFailed {
					for {
						err := downloadWithProgressBar(&ipsw, downloadPath)

						if err == nil {
							break
						}
					}
				}
			} else if err != nil && !os.IsNotExist(err) {
				log.Printf("Error reading download path: %s, err: %s", downloadPath, err)
			}
		}
	}
}

func downloadWithProgressBar(ipsw *api.Firmware, downloadPath string) error {
	filename := filepath.Base(ipsw.URL)

	log.Printf("Downloading %s (%s)", filename, humanize.Bytes(ipsw.Filesize))

	bar := pb.New(int(ipsw.Filesize)).SetUnits(pb.U_BYTES)
	bar.Start()

	checksum, err := download(ipsw.URL, downloadPath, bar, func(n, downloaded int, total int64) {
		downloadedSize += uint64(n)
	})

	bar.Finish()

	if err != nil {
		log.Printf("Error while downloading %s, err: %s", filename, err)
		return err
	} else if checksum != ipsw.SHA1Sum {
		log.Printf("File: %s failed checksum (wanted: %s, got: %s)", filename, ipsw.SHA1Sum, checksum)
		return errors.New("checksum incorrect")
	}

	return nil
}

func parseDownloadDirectory(fw *api.Firmware, identifier string) (string, error) {
	directoryBuffer := new(bytes.Buffer)

	t, err := template.New("firmware").Parse(downloadDirectoryTemplate)

	if err != nil {
		return "", err
	}

	// add the identifier, for simplicity
	fw.Identifier = identifier

	err = t.Execute(directoryBuffer, fw)

	if err != nil {
		return "", nil
	}

	return directoryBuffer.String(), err
}

func verify(location string, expectedSHA1sum string) (bool, error) {
	file, err := os.Open(location)

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

	return expectedSHA1sum == hex.EncodeToString(bs), nil
}

func download(url string, location string, writer io.Writer, callback func(n, downloaded int, total int64)) (string, error) {
	out, err := os.Create(location)

	if err != nil {
		return "", err
	}

	defer out.Close()

	h := sha1.New()
	mw := io.MultiWriter(out, h, writer)

	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	buf := make([]byte, 128*1024)

	downloaded := 0

	for {
		if n, err := resp.Body.Read(buf); (err == nil || err == io.EOF) && n > 0 {
			_, err = mw.Write(buf[:n])

			if err != nil {
				return "", err
			}

			downloaded += n

			if callback != nil {
				callback(n, downloaded, resp.ContentLength)
			}
		} else if err != nil && err != io.EOF {
			return "", err
		} else {
			break
		}
	}

	return hex.EncodeToString(h.Sum(nil)), err
}
