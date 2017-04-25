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
	"github.com/dustin/go-humanize"
)

var (
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

	body, err := getFirmwaresJSON()

	if err != nil {
		log.Fatalf("Unable to retrieve firmware information, err: %s", err)
	}

	for identifier, device := range body.Devices {
		if specifiedDevice != "" && identifier != specifiedDevice {
			continue
		}

		totalDeviceCount++

		for _, ipsw := range device.Firmwares {
			if downloadSigned && !ipsw.Signed {
				continue
			}

			directory, err := parseDownloadDirectory(ipsw, identifier)

			if err != nil {
				log.Printf("Unable to parse download directory, err: %s", err)
				continue
			}

			downloadPath := filepath.Join(directory, ipsw.Filename)

			if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
				totalFirmwareCount++
				totalFirmwareSize += ipsw.Size
			}
		}
	}

	if !verifyIntegrity {
		log.Printf("Downloading: %v IPSW files for %v device(s) (%v)", totalFirmwareCount, totalDeviceCount, humanize.Bytes(totalFirmwareSize))
	}

	for identifier, device := range body.Devices {
		if specifiedDevice != "" && identifier != specifiedDevice {
			continue
		}

		if !verifyIntegrity {
			log.Printf("Downloading %d firmwares for %s", len(device.Firmwares), device.Name)
		}

		for _, ipsw := range device.Firmwares {
			if downloadSigned && !ipsw.Signed {
				continue
			}

			directory, err := parseDownloadDirectory(ipsw, identifier)

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

			downloadPath := filepath.Join(directory, ipsw.Filename)

			_, err = os.Stat(downloadPath)

			if os.IsNotExist(err) && !verifyIntegrity {
				for {
					err := downloadWithProgressBar(ipsw, downloadPath)

					if err == nil || !reDownloadOnVerificationFailed {
						break
					}
				}
			} else if err == nil && verifyIntegrity {
				fileOK, err := verify(downloadPath, ipsw.SHA1)

				if err != nil {
					log.Printf("Error verifying: %s, err: %s", ipsw.Filename, err)
				}

				if fileOK {
					log.Printf("%s verified successfully", ipsw.Filename)
					continue
				}

				log.Printf("%s did not verify successfully", ipsw.Filename)

				if reDownloadOnVerificationFailed {
					for {
						err := downloadWithProgressBar(ipsw, downloadPath)

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

func downloadWithProgressBar(ipsw *Firmware, downloadPath string) error {
	log.Printf("Downloading %s (%s)", ipsw.Filename, humanize.Bytes(ipsw.Size))

	bar := pb.New(int(ipsw.Size)).SetUnits(pb.U_BYTES)
	bar.Start()

	checksum, err := download(ipsw.URL, downloadPath, bar, func(n, downloaded int, total int64) {
		downloadedSize += uint64(n)
	})

	bar.Finish()

	if err != nil {
		log.Printf("Error while downloading %s, err: %s", ipsw.Filename, err)
		return err
	} else if checksum != ipsw.SHA1 {
		log.Printf("File: %s failed checksum (wanted: %s, got: %s)", ipsw.Filename, ipsw.SHA1, checksum)
		return errors.New("checksum incorrect")
	}

	return nil
}

func parseDownloadDirectory(fw *Firmware, identifier string) (string, error) {
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
