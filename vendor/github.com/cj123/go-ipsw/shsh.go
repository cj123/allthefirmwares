package ipsw

import (
	"encoding/json"
	"io/ioutil"
)

type SHSHJSON map[Identifier]*SHSHDevice

type SHSHDevice struct {
	BoardConfig string           `json:"board"`
	Identifier  string           `json:"model"`
	CPID        int              `json:"cpid,string"`
	BDID        int              `json:"bdid,string"`
	Platform    string           `json:"platform"`
	Firmwares   []*SigningStatus `json:"firmwares"`
}

// Firmware is a change to signing window firmware
type SigningStatus struct {
	BuildID string `json:"build"`
	Version string `json:"version"`
	Signing bool   `json:"signing"`
	Started string `json:"started"`
	Stopped string `json:"stopped"`
}

// NewSHSHJSON returns a new instance of SHSHJSON
func NewSHSHJSON(sourceURL string) (SHSHJSON, error) {
	resp, err := DefaultClient.Get(sourceURL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	document, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var shsh SHSHJSON

	err = json.Unmarshal(document, &shsh)

	if err != nil {
		return nil, err
	}

	return shsh, err
}
