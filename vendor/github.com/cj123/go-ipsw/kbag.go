package ipsw

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	NorContainer   ImageContainer = 0x696D6733 // img3
	Img3Container  ImageContainer = 0x496D6733 // Img3
	X8900Container ImageContainer = 0x30303938 // 8900
	Img2CVontainer ImageContainer = 0x494D4732 // IMG2

	DataElement ElementType = 0x44415441 // DATA
	TypeElement ElementType = 0x54595045 // TYPE
	KbagElement ElementType = 0x4B424147 // KBAG
	ShshElement ElementType = 0x53485348 // SHSH
	CertElement ElementType = 0x43455254 // CERT
	ChipElement ElementType = 0x43484950 // CHIP
	ProdElement ElementType = 0x50524F44 // PROD
	SdomElement ElementType = 0x53444F4D // SDOM
	BordElement ElementType = 0x424F5244 // BORD
	SepoElement ElementType = 0x5345504F // SEPO
	EcidElement ElementType = 0x45434944 // ECID
)

type ImageContainer int
type ElementType uint32

type ImageHeader struct {
	Signature, FullSize, DataSize, ShshOffset, ImageType uint32
}

type ImageElementHeader struct {
	Signature, FullSize, DataSize uint32
}

type ImageKbagElement struct {
	Header       ImageElementHeader
	State, IType uint32
	IV           [16]byte
	Key          [32]byte
}

func FindElement(data *bufio.Reader, signature ElementType, out interface{}) error {
	for {
		b, err := data.Peek(4)

		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			break
		}

		v := binary.LittleEndian.Uint32(b)
		if v == uint32(signature) {
			return binary.Read(data, binary.LittleEndian, out)
		}

		data.Discard(4)
	}

	return fmt.Errorf("element with signature: %d not found", signature)
}

// KBag finds the kbag for a byte array.
func KBag(data *bufio.Reader) (string, error) {
	var elem ImageKbagElement

	err := FindElement(data, KbagElement, &elem)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", append(elem.IV[:], elem.Key[:]...)), nil
}
