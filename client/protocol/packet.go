package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Message type constants for different packet types.
const (
	MsgBet   = 0x01 // BetPacket message type
	MsgReply = 0x02 // ReplyPacket message type
	MsgError = 0x03 // ErrorPacket message type
)

// Error code constants for ErrorPacket.
const (
	ErrInvalidPacket  = 0x01 // Invalid packet structure error
	ErrInvalidBetData = 0x02 // Invalid bet data error
)

/*
Header represents the common header for all protocol packets.

The header is structured as:
- 1 Byte for MessageType
- 4 Bytes for PayloadLength (big-endian)
*/
type Header struct {
	MessageType   uint8
	PayloadLength uint32
}

// HeaderSize defines the fixed size of the packet header in bytes.
const HeaderSize = 5

// DeserializeHeader converts a byte slice into a Header struct.
// Returns an error if the byte slice length doesn't match HeaderSize.
func DeserializeHeader(headerBytes []byte) (*Header, error) {
	if len(headerBytes) != HeaderSize {
		return nil, errors.New("invalid header size")
	}

	return &Header{
		MessageType:   headerBytes[0],
		PayloadLength: binary.BigEndian.Uint32(headerBytes[1:5]),
	}, nil
}

// Write serializes the Header to an io.Writer.
func (h *Header) Write(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, h.MessageType); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, h.PayloadLength)
}

// Packet interface defines the common methods for all protocol packet types.
type Packet interface {
	Type() uint8              // Returns the message type identifier
	Encode(w io.Writer) error // Serializes the packet to bytes
}

// WritePacket writes a complete packet (header + payload) to an io.Writer.
func WritePacket(w io.Writer, packet Packet) error {
	var payloadBuf bytes.Buffer
	if err := packet.Encode(&payloadBuf); err != nil {
		return err
	}
	payload := payloadBuf.Bytes()

	header := Header{
		MessageType:   packet.Type(),
		PayloadLength: uint32(len(payload)),
	}
	if err := header.Write(w); err != nil {
		return err
	}

	_, err := w.Write(payload)
	return err
}

// DeserializePayload converts a byte slice into the appropriate Packet type
// based on the messageType. Returns an error for unknown message types.
func DeserializePayload(messageType uint8, payloadBytes []byte) (Packet, error) {
	payloadReader := bytes.NewReader(payloadBytes)

	switch messageType {
	case MsgBet:
		return DecodeBetPacket(payloadReader)
	case MsgReply:
		return DecodeReplyPacket(payloadReader)
	case MsgError:
		return DecodeErrorPacket(payloadReader)
	default:
		return nil, fmt.Errorf("unknown message type: %d", messageType)
	}
}

/*
BetPacket represents a bet submission from a client agency.

The packet structure is:
- 1 Byte for AgencyID
- Bet struct (see Bet documentation for structure)
*/
type BetPacket struct {
	AgencyID uint8
	Bet      Bet
}

// Type returns the message type identifier for BetPacket.
func (p *BetPacket) Type() uint8 { return MsgBet }

// Encode serializes the BetPacket to an io.Writer.
func (p *BetPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.AgencyID); err != nil {
		return err
	}
	return p.Bet.Encode(w)
}

// DecodeBetPacket deserializes a BetPacket from an io.Reader.
func DecodeBetPacket(r io.Reader) (*BetPacket, error) {
	var agencyID uint8
	if err := binary.Read(r, binary.BigEndian, &agencyID); err != nil {
		return nil, err
	}

	var bet Bet
	if err := bet.Decode(r); err != nil {
		return nil, err
	}

	return &BetPacket{AgencyID: agencyID, Bet: bet}, nil
}

// NewBetPacket creates a new BetPacket from string agency ID and Bet.
// Returns an error if the agency ID cannot be converted to uint8.
func NewBetPacket(id string, bet Bet) (*BetPacket, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, err
	}
	return &BetPacket{
		AgencyID: uint8(n),
		Bet:      bet,
	}, nil
}

/*
ReplyPacket represents a server response to a client request.

The packet structure is:
- 4 Bytes for DoneCount (big-endian)
- 1 Byte for message length
- N Bytes for message string
*/
type ReplyPacket struct {
	DoneCount uint32
	Message   string
}

// Type returns the message type identifier for ReplyPacket.
func (p *ReplyPacket) Type() uint8 { return MsgReply }

// Encode serializes the ReplyPacket to an io.Writer.
func (p *ReplyPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.DoneCount); err != nil {
		return err
	}

	msgBytes := []byte(p.Message)
	if len(msgBytes) > 255 {
		return errors.New("message too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint8(len(msgBytes))); err != nil {
		return err
	}
	_, err := w.Write(msgBytes)
	return err
}

// DecodeReplyPacket deserializes a ReplyPacket from an io.Reader.
func DecodeReplyPacket(r io.Reader) (*ReplyPacket, error) {
	var doneCount uint32
	if err := binary.Read(r, binary.BigEndian, &doneCount); err != nil {
		return nil, err
	}

	var msgLen uint8
	if err := binary.Read(r, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	msgBytes := make([]byte, msgLen)
	if _, err := io.ReadFull(r, msgBytes); err != nil {
		return nil, err
	}

	return &ReplyPacket{
		DoneCount: doneCount,
		Message:   string(msgBytes),
	}, nil
}

/*
ErrorPacket represents an error response from the server.

The packet structure is:
- 1 Byte for ErrorCode
- 1 Byte for message length
- N Bytes for message string
*/
type ErrorPacket struct {
	ErrorCode uint8
	Message   string
}

// Type returns the message type identifier for ErrorPacket.
func (p *ErrorPacket) Type() uint8 { return MsgError }

// Encode serializes the ErrorPacket to an io.Writer.
func (p *ErrorPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.ErrorCode); err != nil {
		return err
	}

	msgBytes := []byte(p.Message)
	if len(msgBytes) > 255 {
		return errors.New("message too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint8(len(msgBytes))); err != nil {
		return err
	}
	_, err := w.Write(msgBytes)
	return err
}

// DecodeErrorPacket deserializes an ErrorPacket from an io.Reader.
func DecodeErrorPacket(r io.Reader) (*ErrorPacket, error) {
	var errorCode uint8
	if err := binary.Read(r, binary.BigEndian, &errorCode); err != nil {
		return nil, err
	}

	var msgLen uint8
	if err := binary.Read(r, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	msgBytes := make([]byte, msgLen)
	if _, err := io.ReadFull(r, msgBytes); err != nil {
		return nil, err
	}

	return &ErrorPacket{
		ErrorCode: errorCode,
		Message:   string(msgBytes),
	}, nil
}

// ErrorFromPacket converts an ErrorPacket into a human-readable error string.
func ErrorFromPacket(e ErrorPacket) string {
	switch e.ErrorCode {
	case ErrInvalidPacket:
		return "INVALID_PACKET"
	case ErrInvalidBetData:
		return "INVALID_BET"
	default:
		return "UNKNOWN_ERROR"
	}
}
