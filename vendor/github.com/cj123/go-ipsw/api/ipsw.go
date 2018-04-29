package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/guregu/null.v3"
)

// ReleaseType represents a release type
type ReleaseType string

const (
	// ReleaseTypeiOS is an iOS release
	ReleaseTypeiOS ReleaseType = "iOS"

	// ReleaseTypeDevice is a device release
	ReleaseTypeDevice ReleaseType = "Device"

	// ReleaseTypeRedsn0w is a redsn0w release
	ReleaseTypeRedsn0w ReleaseType = "redsn0w"

	// ReleaseTypePwnageTool is a PwnageTool release
	ReleaseTypePwnageTool ReleaseType = "PwnageTool"

	// ReleaseTypeiTunes is an iTunes release
	ReleaseTypeiTunes ReleaseType = "iTunes"

	// ReleaseTypeiOSOTA is an OTA release
	ReleaseTypeiOSOTA ReleaseType = "iOS OTA"

	// ReleaseTypewatchOS is a watchOS release
	ReleaseTypewatchOS ReleaseType = "watchOS"

	// ReleaseTypeSigning is a signing change to an iOS firmware
	ReleaseTypeSigning ReleaseType = "shsh"
)

type BaseDevice struct {
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	BoardConfig string `json:"BoardConfig"`
	Platform    string `json:"platform"`
	CPID        int    `json:"cpid"`
	BDID        int    `json:"bdid"`
}

// Device is an iOS device released by Apple, and all available IPSW files for it.
type Device struct {
	BaseDevice
	Firmwares []Firmware `json:"firmwares"`
}

// Device is an iOS device released by Apple, and all available OTA files for it.
type OTADevice struct {
	BaseDevice
	Firmwares []OTAFirmware `json:"firmwares"`
}

// Firmware represents everything known by IPSW Downloads about an IPSW file
type Firmware struct {
	Identifier  string    `json:"identifier"`
	Version     string    `json:"version"`
	Device      string    `json:"device"`
	BuildID     string    `json:"buildid"`
	SHA1Sum     string    `json:"sha1sum"`
	MD5Sum      string    `json:"md5sum"`
	Filesize    uint64    `json:"filesize"`
	UploadDate  null.Time `json:"uploaddate"`
	ReleaseDate null.Time `json:"releasedate"`
	URL         string    `json:"url"`
	Signed      bool      `json:"signed"`
}

// OTAFirmware represents an "over-the-air" firmware file
type OTAFirmware struct {
	Firmware
	PrerequisiteVersion string `json:"prerequisiteversion"`
	PrerequisiteBuildID string `json:"prerequisitebuildid"`
	ReleaseType         string `json:"releasetype"`
}

// ITunes represents an iTunes download.
type ITunes struct {
	Version         string    `json:"version"`
	UploadDate      null.Time `json:"uploaddate"`
	ReleaseDate     null.Time `json:"releasedate"`
	URL             string    `json:"url"`
	SixtyFourBitURL string    `json:"64biturl"`
}

type ReleasesByDate struct {
	Date     string
	Releases []Release
}

// Release is an iOS/iTunes/... release detected by IPSW Downloads
type Release struct {
	Name  string      `json:"name"`
	Date  time.Time   `json:"date"`
	Count int         `json:"count"`
	Type  ReleaseType `json:"type"`
}

// FirmwareInfo is a representation of keys information known by IPSW Downloads
type FirmwareInfo struct {
	Identifier           string `json:"identifier"`
	BuildID              string `json:"buildid"`
	CodeName             string `json:"codename"`
	Baseband             string `json:"baseband,omitempty"`
	UpdateRamdiskExists  bool   `json:"updateramdiskexists"`
	RestoreRamdiskExists bool   `json:"restoreramdiskexists"`

	Keys []FirmwareKey `json:"keys,omitempty"`
}

// FirmwareKey is a key/iv combo for an individual firmware file
type FirmwareKey struct {
	Image    string    `json:"image"`
	Filename string    `json:"filename"`
	KBag     string    `json:"kbag"`
	Key      string    `json:"key"`
	IV       string    `json:"iv"`
	Date     time.Time `json:"date"`
}

type IPSWClient struct {
	client *http.Client
	base   string
}

// NewIPSWClient creates an IPSWClient. If client == nil, http.DefaultClient is used.
func NewIPSWClient(apiBase string, client *http.Client) *IPSWClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &IPSWClient{
		base:   apiBase,
		client: client,
	}
}

// makeRequest makes the http request to a given endpoint
// note: makeRequest does not call resp.Body.Close(), this must be done manually
func (c *IPSWClient) makeRequest(url string, headers map[string]string) (*http.Response, error) {
	request, err := http.NewRequest("GET", c.base+url, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Add("Accept", "application/json")

	for key, val := range headers {
		request.Header.Add(key, val)
	}

	res, err := c.client.Do(request)

	if err != nil {
		return nil, err
	}

	return res, err
}

func parseJSON(res *http.Response, output interface{}) error {
	return json.NewDecoder(res.Body).Decode(&output)
}

func (c *IPSWClient) Devices(onlyShowDevicesWithKeys bool) ([]BaseDevice, error) {
	var devices []BaseDevice

	requestURL := "/devices"

	if onlyShowDevicesWithKeys {
		requestURL += "?keysOnly=true"
	}

	resp, err := c.makeRequest(requestURL, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &devices)

	return devices, err
}

func (c *IPSWClient) DeviceInformation(identifier string) (*Device, error) {
	var device *Device

	resp, err := c.makeRequest("/device/"+identifier, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &device)

	return device, err
}

func (c *IPSWClient) OTADeviceInformation(identifier string) (*OTADevice, error) {
	var device *OTADevice

	resp, err := c.makeRequest("/device/"+identifier+"?type=ota", nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &device)

	return device, err
}

func (c *IPSWClient) IPSWInformation(identifier, buildid string) (*Firmware, error) {
	var fw *Firmware

	resp, err := c.makeRequest(fmt.Sprintf("/ipsw/%s/%s", identifier, buildid), nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &fw)

	return fw, err
}

func (c *IPSWClient) OTAInformation(identifier, buildid, prerequisite string) (*OTAFirmware, error) {
	var fw *OTAFirmware

	resp, err := c.makeRequest(fmt.Sprintf("/ota/%s/%s", identifier, buildid), nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &fw)

	return fw, err
}

func (c *IPSWClient) IPSWsForVersion(version string) ([]Firmware, error) {
	var fws []Firmware

	resp, err := c.makeRequest("/ipsw/"+version, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &fws)

	return fws, err
}

func (c *IPSWClient) OTAsForVersion(version string) ([]OTAFirmware, error) {
	var fws []OTAFirmware

	resp, err := c.makeRequest("/ota/"+version, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &fws)

	return fws, err
}

func (c *IPSWClient) ITunes(platform string) ([]ITunes, error) {
	var itunes []ITunes

	resp, err := c.makeRequest("/itunes/"+platform, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &itunes)

	return itunes, err
}

func (c *IPSWClient) KeysList(identifier string) ([]FirmwareInfo, error) {
	var info []FirmwareInfo

	resp, err := c.makeRequest("/keys/device/"+identifier, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &info)

	return info, err
}

func (c *IPSWClient) KeysForIPSW(identifier, buildid string) (*FirmwareInfo, error) {
	var info *FirmwareInfo

	resp, err := c.makeRequest(fmt.Sprintf("/keys/ipsw/%s/%s", identifier, buildid), nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &info)

	return info, err
}

func (c *IPSWClient) ReleaseInformation() ([]ReleasesByDate, error) {
	var releases []ReleasesByDate

	resp, err := c.makeRequest("/releases", nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &releases)

	return releases, err
}

type modelResponse struct {
	Identifier string `json:"identifier"`
}

func (c *IPSWClient) IdentifyModel(model string) (string, error) {
	var r modelResponse

	resp, err := c.makeRequest("/model/"+model, nil)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	err = parseJSON(resp, &r)

	return r.Identifier, err
}

func (c *IPSWClient) URL(identifier, buildid string) (string, error) {
	fw, err := c.IPSWInformation(identifier, buildid)

	if err != nil {
		return "", err
	}

	return fw.URL, nil
}
