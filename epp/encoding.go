package epp

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
)

// EPPHeader is appended to XML messages that are sent to the server.
var EPPHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n"

// minimumTokenSize represents the minimum number of characters for a token to be
// decoded.
const minimumTokenSize = 4

// WireSplit is used by the bufio.Scanner to split epp messages apart
// before they are unmarshalled.
func WireSplit(data []byte, _ bool) (advance int, token []byte, err error) {
	avilDataLen := uint32(len(data))

	if avilDataLen >= minimumTokenSize {
		var msgSize uint32

		lenBuf := bytes.NewBuffer(data[:minimumTokenSize])

		err = binary.Read(lenBuf, binary.BigEndian, &msgSize)
		if avilDataLen >= msgSize && err == nil {
			advance = int(msgSize)
			token = data[4:msgSize]
			err = nil

			return
		}

		return 0, nil, nil
	}

	return 0, nil, nil
}

// EncodeEPP takes an EPP object and converts it into a stream of bytes
// that can be written to the wire to send an EPP message.
func (e Epp) EncodeEPP() (buf []byte, err error) {
	var buffer bytes.Buffer

	var xmlBytes string

	xmlBytes, err = e.ToString()

	if err == nil {
		hdrMsg := append([]byte(EPPHeader), xmlBytes...)
		msgSize := uint32(len(hdrMsg)) + minimumTokenSize

		err = binary.Write(&buffer, binary.BigEndian, msgSize)
		if err == nil {
			hdrMsgStr := string(hdrMsg)
			buffer.Grow(len(hdrMsg))
			_, err = buffer.WriteString(hdrMsgStr)
		}
	}

	if err != nil {
		return buffer.Bytes(), fmt.Errorf("error encoding EPP message: %w", err)
	}

	return buffer.Bytes(), nil
}

// UnmarshalMessage takes a string of bytes and attempts to unmarshal
// the data into an EPP object. If an error occurs it is returned
// otherwise the error return value will be nil.
func UnmarshalMessage(message []byte) (output Epp, err error) {
	err = xml.Unmarshal(message, &output)

	return
}
