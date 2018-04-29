package ipsw

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/cj123/go-ipsw/api"
	"howett.net/plist"
)

const (
	BuildManifestFilename = "BuildManifest.plist"
	RestoreFilename       = "Restore.plist"
)

type BuildManifest struct {
	BuildIdentities       []BuildIdentity
	SupportedProductTypes []string
	ProductVersion        string
	ProductBuildVersion   string
}

type Restore struct {
	DeviceClass           string
	Devices               []*Device `plist:"DeviceMap"`
	ProductBuildVersion   string
	ProductType           Identifier
	ProductVersion        string
	SupportedProductTypes []Identifier
}

func (r *Restore) DeviceByIdentifier(identifier Identifier) (*Device, error) {
	var device *Device

	if len(r.Devices) == 1 {
		device = r.Devices[0]
	} else {
		for deviceIndex, restoreDevice := range r.SupportedProductTypes {
			if restoreDevice == identifier {
				device = r.Devices[deviceIndex]
				break
			}
		}
	}

	if device == nil {
		if r.ProductType == identifier {
			device = r.Devices[0]
		} else {
			return nil, fmt.Errorf("ipsw: unable to find identifier: %s in Restore.plist", identifier)
		}
	}

	return device, nil
}

type Device struct {
	Identifier  Identifier
	BDID        int
	BoardConfig string
	CPID        int
	Platform    string
	SCEP        int
}

type BuildIdentity struct {
	ApChipID                           string // these are ints really
	ApBoardID                          string
	ApSecurityDomain                   string
	BbChipID                           string
	BbProvisioningManifestKeyHash      []byte
	BbActivationManifestKeyHash        []byte
	BbCalibrationManifestKeyHash       []byte
	BbFactoryActivationManifestKeyHash []byte
	BbFDRSecurityKeyHash               []byte
	Info                               BuildIdentityInfo
	Manifest                           BuildIdentityManifest `plist:"Manifest"`
	// RawManifest                        interface{}           `plist:"Manifest"`
	UniqueBuildID []byte
	BbSkeyId      []byte
}

type BuildIdentityInfo struct {
	BuildNumber string
	BuildTrain  string
	DeviceClass string
}

type BuildIdentityManifest map[string]Manifest

type Manifest struct {
	Info ManifestInfo
}

type ManifestInfo struct {
	Path string
}

type IPSW struct {
	Identifier  string
	BuildID     string
	Resource    string
	manifest    *BuildManifest
	rawManifest map[string]interface{}
	restore     *Restore
	headers     http.Header
}

func NewIPSW(identifier, build, resource string) *IPSW {
	return &IPSW{
		Identifier: identifier,
		BuildID:    build,
		Resource:   resource,
	}
}

func NewIPSWWithIdentifierBuild(client *api.IPSWClient, identifier, build string) (*IPSW, error) {
	resource, err := client.URL(identifier, build)

	if err != nil {
		return nil, err
	} else if resource == "" {
		return nil, errors.New("ipsw: firmware not found (potentially beta)")
	}

	return NewIPSW(identifier, build, resource), nil
}

func (i *IPSW) PlistFromZip(name string, out interface{}) error {
	buf := new(bytes.Buffer)
	writer := bufio.NewWriter(buf)

	err := DownloadFile(i.Resource, name, writer)

	if err != nil {
		return err
	}

	_, err = plist.Unmarshal(buf.Bytes(), out)

	return err
}

func (i *IPSW) Headers() (http.Header, error) {
	if i.headers != nil {
		return i.headers, nil
	}

	res, err := DefaultClient.Get(i.Resource)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	i.headers = res.Header

	return res.Header, err
}

func (i *IPSW) BuildManifest() (*BuildManifest, error) {
	if i.manifest != nil {
		return i.manifest, nil
	}

	var manifest BuildManifest

	err := i.PlistFromZip(BuildManifestFilename, &manifest)

	if err != nil {
		return nil, err
	}

	i.manifest = &manifest

	return &manifest, err
}

func (i *IPSW) RawManifest() (map[string]interface{}, error) {
	if i.rawManifest != nil {
		return i.rawManifest, nil
	}

	var manifest map[string]interface{}

	err := i.PlistFromZip(BuildManifestFilename, &manifest)

	if err != nil {
		return nil, err
	}

	i.rawManifest = manifest

	return manifest, err
}

func (i *IPSW) RestorePlist() (*Restore, error) {
	if i.restore != nil {
		return i.restore, nil
	}

	var restore Restore

	err := i.PlistFromZip(RestoreFilename, &restore)

	if err != nil {
		return nil, err
	}

	i.restore = &restore

	return &restore, err
}

var basebandRegex = regexp.MustCompile("[0-9]{2}.[0-9]{2}.[0-9]{2}")

func (i IPSW) Baseband() (string, error) {
	manifest, err := i.BuildManifest()

	if err != nil {
		return "", err
	}

	productIndex := -1

	for index, productType := range manifest.SupportedProductTypes {
		if productType == i.Identifier {
			productIndex = index
		}
	}

	if productIndex == -1 {
		return "", errors.New("ipsw: unable to find identifier in given IPSW")
	}

	baseband, ok := manifest.BuildIdentities[productIndex].Manifest["BasebandFirmware"]

	if !ok {
		return "", errors.New("ipsw: baseband not found in IPSW")
	}

	return basebandRegex.FindString(baseband.Info.Path), nil
}
