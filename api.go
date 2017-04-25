package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	apiURL    = "https://api.ipsw.me/v3/firmwares.json"
	userAgent = "allthefirmwares/1.0"
)

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
	Size        uint64 `json:"size"`
	ReleaseDate string `json:"releasedate,omitempty"`
	UploadDate  string `json:"uploaddate"`
	URL         string `json:"url"`
	Signed      bool   `json:"signed"`
	Filename    string `json:"filename"`
}

type APIJSON struct {
	Devices map[string]*Device `json:"devices"`
}

// get the JSON from API_URL and parse it
func getFirmwaresJSON() (parsed *APIJSON, err error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	response, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, &parsed)

	if err != nil {
		return nil, err
	}

	return parsed, err
}
