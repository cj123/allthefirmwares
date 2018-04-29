package ipsw

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"

	"howett.net/plist"
)

const OTABuildManifestFilename = "AssetData/boot/BuildManifest.plist"

type OTAXML struct {
	Assets []*OTAFirmware
}

// Firmware release
type OTAFirmware struct {
	BuildID               string       `plist:"Build"`
	InstallationSize      string       `plist:"InstallationSize"`
	Version               string       `plist:"OSVersion"`
	PrerequisiteBuild     string       `plist:"PrerequisiteBuild"`
	PrerequisiteOSVersion string       `plist:"PrerequisiteOSVersion"`
	ReleaseType           string       `plist:"ReleaseType"`
	DocumentationID       string       `plist:"SUDocumentationID"`
	SupportedDevices      []Identifier `plist:"SupportedDevices"`
	DownloadSize          int          `plist:"_DownloadSize"`
	UnarchivedSize        int          `plist:"_UnarchivedSize"`
	BaseURL               string       `plist:"__BaseURL"`
	RelativePath          string       `plist:"__RelativePath"`
	MarketingVersion      string       `plist:"MarketingVersion"` // for watches
}

func (o *OTAFirmware) GetURL() string {
	return o.BaseURL + o.RelativePath
}

// NewOTAXML generates a new OTA XML struct
func NewOTAXML(src string) (*OTAXML, error) {
	resp, err := DefaultClient.Get(src)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	document, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var ota OTAXML

	_, err = plist.Unmarshal(document, &ota)

	if err != nil {
		return nil, err
	}

	return &ota, err
}

type OTABuildManifest struct {
	BuildIdentities       []*OTABuildIdentity `plist:"BuildIdentities"`
	SupportedProductTypes []Identifier        `plist:"SupportedProductTypes"`
	ProductVersion        string              `plist:"ProductVersion"`
	ProductBuildVersion   string              `plist:"ProductBuildVersion"`
}

type OTABuildIdentity struct {
	ApChipID  string                `plist:"ApChipID"`
	ApBoardID string                `plist:"ApBoardID"`
	Manifest  map[string]*Image     `plist:"Manifest"`
	Info      *OTABuildIdentityInfo `plist:"Info"`
}

type Image struct {
	Info struct {
		Path string `plist:"Path"`
	} `plist:"Info"`
}

type OTABuildIdentityInfo struct {
	DeviceClass string `plist:"DeviceClass"`
}

type OTAZip struct {
	*IPSW

	manifest *OTABuildManifest
}

func NewOTAZip(identifier, build, resource string) *OTAZip {
	return &OTAZip{
		IPSW: &IPSW{
			Identifier: identifier,
			BuildID:    build,
			Resource:   resource,
		},
	}
}

func (z *OTAZip) BuildManifest() (*OTABuildManifest, error) {
	if z.manifest != nil {
		return z.manifest, nil
	}

	var manifest OTABuildManifest

	err := z.PlistFromZip(OTABuildManifestFilename, &manifest)

	z.manifest = &manifest

	return &manifest, err
}

func (m *OTABuildManifest) DeviceByIdentifier(identifier Identifier) (*Device, error) {
	var identity *OTABuildIdentity

	if len(m.BuildIdentities) < 1 {
		return nil, errors.New("invalid build identities")
	} else if len(m.SupportedProductTypes) != len(m.BuildIdentities) {
		// just use 0
		identity = m.BuildIdentities[0]
	} else {
		for index, device := range m.SupportedProductTypes {
			if device != identifier {
				continue
			}

			identity = m.BuildIdentities[index]
		}
	}

	chipID, err := strconv.ParseInt(identity.ApChipID, 0, 0)

	if err != nil {
		return nil, err
	}

	boardID, err := strconv.ParseInt(identity.ApBoardID, 0, 0)

	if err != nil {
		return nil, err
	}

	device := &Device{
		Identifier:  identifier,
		BoardConfig: identity.Info.DeviceClass,
		CPID:        int(chipID),
		BDID:        int(boardID),
	}

	// @TODO this is a hack and should be removed...
	if val, ok := identity.Manifest["AppleLogo"]; ok {
		device.Platform = findPlatformFromPath(val.Info.Path)
	}

	return device, err
}

func findPlatformFromPath(path string) string {
	parts := strings.Split(path, ".")
	num := len(parts)

	if num > 2 {
		return parts[num-2]
	}

	return ""
}
