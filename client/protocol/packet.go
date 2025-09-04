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
	MsgBetStart     = 0x01 // BetStartPacket message type
	MsgBet          = 0x02 // BetPacket message type
	MsgBetFinish    = 0x03 // BetFinishPacket message type
	MsgReply        = 0x04 // ReplyPacket message type
	MsgGetWinners   = 0x05 // GetWinnersPacket message type
	MsgReplyWinners = 0x06 // ReplyWinnersPacket message type
	MsgError        = 0x07 // ErrorPacket message type
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
// based on the messageType. Returns an error for unknown or invalid message types.
func DeserializePayload(messageType uint8, payloadBytes []byte) (Packet, error) {
	payloadReader := bytes.NewReader(payloadBytes)

	switch messageType {
	case MsgReply:
		return DecodeReplyPacket(payloadReader)
	case MsgReplyWinners:
		return DecodeReplyWinnersPacket(payloadReader)
	case MsgError:
		return DecodeErrorPacket(payloadReader)
	default:
		return nil, fmt.Errorf("unknown/invalid message type from server: %d", messageType)
	}
}

/*
BetStartPacket represents an agency message to the server to make it start listening for BetPacket.
The packet structure is:
- 1 Byte for agency ID.
*/
type BetStartPacket struct {
	AgencyID uint8
}

// Type returns the message type identifier for BetStartPacket.
func (b *BetStartPacket) Type() uint8 { return MsgBetStart }

// Encode serializes the BetStartPacket to an io.Writer.
func (b *BetStartPacket) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, b.AgencyID)
}

// NewBetStartPacket creates a new BetStartPacket from string agency ID.
// Returns an error if the agency ID cannot be converted to uint8.
func NewBetStartPacket(id string) (*BetStartPacket, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, err
	}
	return &BetStartPacket{
		AgencyID: uint8(n),
	}, nil
}

/*
BetPacket represents a bet submission from a client agency.

The packet structure is:
- 1 Byte for AgencyID
- 4 Bytes for bet_n
- Bet struct list (see Bet documentation for structure)
*/
type BetPacket struct {
	AgencyID uint8
	Bets     []Bet
}

// Type returns the message type identifier for BetPacket.
func (p *BetPacket) Type() uint8 { return MsgBet }

// Encode serializes the BetPacket to an io.Writer.
func (p *BetPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.AgencyID); err != nil {
		return err
	}

	betN := uint32(len(p.Bets))
	if err := binary.Write(w, binary.BigEndian, betN); err != nil {
		return err
	}

	for i, bet := range p.Bets {
		if err := bet.Encode(w); err != nil {
			return fmt.Errorf("writing bet %d: %w", i, err)
		}
	}
	return nil
}

// NewBetPacket creates a new BetPacket from string agency ID and a Bet slice.
// Returns an error if the agency ID cannot be converted to uint8.
func NewBetPacket(id string, bets []Bet) (*BetPacket, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, err
	}
	return &BetPacket{
		AgencyID: uint8(n),
		Bets:     bets,
	}, nil
}

/*
BetFinishPacket represents an agency message to the server in order to signal no more bets.
The packet structure is:
- 1 Byte for agency ID.
*/
type BetFinishPacket struct {
	AgencyID uint8
}

// Type returns the message type identifier for BetFinishPacket.
func (b *BetFinishPacket) Type() uint8 { return MsgBetFinish }

// Encode serializes the BetFinishPacket to an io.Writer.
func (b *BetFinishPacket) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, b.AgencyID)
}

// NewBetFinishPacket creates a new BetFinishPacket from string agency ID.
// Returns an error if the agency ID cannot be converted to uint8.
func NewBetFinishPacket(id string) (*BetFinishPacket, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, err
	}
	return &BetFinishPacket{
		AgencyID: uint8(n),
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
func (p *ReplyPacket) Encode(_ io.Writer) error {
	return errors.New("client has no need to send ReplyPacket")
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
GetWinnersPacket represents an agency message to the server in order to query its winners.

The packet structure is:
- 1 Byte for agency ID.
*/
type GetWinnersPacket struct {
	AgencyID uint8
}

// Type returns the message type identifier for GetWinnersPacket.
func (g *GetWinnersPacket) Type() uint8 { return MsgGetWinners }

// Encode serializes the GetWinnersPacket to an io.Writer.
func (g *GetWinnersPacket) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, g.AgencyID)
}

// NewGetWinnersPacket creates a new GetWinnersPacket from string agency ID.
// Returns an error if the agency ID cannot be converted to uint8.
func NewGetWinnersPacket(id string) (*GetWinnersPacket, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, err
	}
	return &GetWinnersPacket{
		AgencyID: uint8(n),
	}, nil
}

/*
ReplyWinnersPacket represents a server response to a client request for winners.

The packet structure is:
- 1 Byte for AgencyID
- 4 Bytes for amount of winners
- N * 4 Bytes for all the winner document numbers (uint32, big-endian)
*/
type ReplyWinnersPacket struct {
	AgencyID uint8
	Winners  []uint32
}

// Type returns the message type identifier for ReplyWinnersPacket.
func (r *ReplyWinnersPacket) Type() uint8 { return MsgReplyWinners }

// Encode serializes the ReplyWinnersPacket to an io.Writer.
func (r *ReplyWinnersPacket) Encode(_ io.Writer) error {
	return errors.New("client has no need to send ReplyWinnersPacket")
}

// DecodeReplyWinnersPacket deserializes a ReplyWinnersPacket from an io.Reader.
func DecodeReplyWinnersPacket(r io.Reader) (*ReplyWinnersPacket, error) {
	var agencyID uint8
	if err := binary.Read(r, binary.BigEndian, &agencyID); err != nil {
		return nil, fmt.Errorf("reading agency ID: %w", err)
	}

	var winnerCount uint32
	if err := binary.Read(r, binary.BigEndian, &winnerCount); err != nil {
		return nil, fmt.Errorf("reading winner count: %w", err)
	}

	winners := make([]uint32, winnerCount)
	for i := uint32(0); i < winnerCount; i++ {
		if err := binary.Read(r, binary.BigEndian, &winners[i]); err != nil {
			return nil, fmt.Errorf("reading winner %d: %w", i, err)
		}
	}

	return &ReplyWinnersPacket{
		AgencyID: agencyID,
		Winners:  winners,
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
func (p *ErrorPacket) Encode(_ io.Writer) error {
	return errors.New("client has no need to send ErrorPacket")
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
