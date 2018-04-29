package ipsw

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	"howett.net/plist"
)

type BuildNumber string

func (b BuildNumber) String() string {
	return string(b)
}

type IndividualBuild struct {
	BuildVersion     BuildNumber
	DocumentationURL string
	FirmwareURL      string
	FirmwareSHA1     string
	ProductVersion   string
}

type BuildInformation struct {
	Restore              *IndividualBuild
	Update               *IndividualBuild
	SameAs               BuildNumber
	OfferRestoreAsUpdate bool
}

type Identifier string

func (i Identifier) String() string {
	return string(i)
}

type VersionWrapper struct {
	MobileDeviceSoftwareVersions map[Identifier]map[BuildNumber]*BuildInformation
}

type iTunesVersionMaster struct {
	MobileDeviceSoftwareVersionsByVersion map[string]*VersionWrapper
}

// Given a device, find a URL from the version master
// the URL itself doesn't matter, so long as it's an IPSW for the device we requested
func (vm *iTunesVersionMaster) GetSoftwareURLFor(identifier string) (string, error) {
	for _, deviceSoftwareVersions := range vm.MobileDeviceSoftwareVersionsByVersion {
		for i, builds := range deviceSoftwareVersions.MobileDeviceSoftwareVersions {
			if i.String() == identifier {
				for _, build := range builds {
					if build.Restore != nil {
						// don't return protected ones if we can avoid it
						if strings.Contains(build.Restore.FirmwareURL, "protected://") {
							continue
						}
						return build.Restore.FirmwareURL, nil
					}
				}
			}
		}
	}

	return "", errors.New("Unable to find identifier")
}

// NewiTunesVersionMaster creates a new iTunesVersionMaster struct, parsed and ready to use
func NewiTunesVersionMaster(url string) (*iTunesVersionMaster, error) {
	resp, err := DefaultClient.Get(fmt.Sprintf("%s?%d", url, rand.Int()))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	document, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	vm := iTunesVersionMaster{}

	_, err = plist.Unmarshal(document, &vm)

	return &vm, err
}
