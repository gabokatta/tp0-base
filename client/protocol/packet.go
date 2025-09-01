package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	MsgBet   = 0x01
	MsgReply = 0x02
	MsgError = 0x03
)

const (
	ReplyStatusOK    = 0x00
	ReplyStatusError = 0x01
)

const (
	ErrInvalidPacket  = 0x01
	ErrInvalidBetData = 0x02
)

type Header struct {
	MessageType   uint8
	PayloadLength uint32
}

const HeaderSize = 5

func DeserializeHeader(headerBytes []byte) (*Header, error) {
	if len(headerBytes) != HeaderSize {
		return nil, errors.New("invalid header size")
	}

	return &Header{
		MessageType:   headerBytes[0],
		PayloadLength: binary.BigEndian.Uint32(headerBytes[1:5]),
	}, nil
}

func (h *Header) Write(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, h.MessageType); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, h.PayloadLength)
}

type Packet interface {
	Type() uint8
	Encode(w io.Writer) error
}

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

type BetPacket struct {
	AgencyID uint8
	Bet      Bet
}

func (p *BetPacket) Type() uint8 { return MsgBet }

func (p *BetPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.AgencyID); err != nil {
		return err
	}
	return p.Bet.Encode(w)
}

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

type ReplyPacket struct {
	Status    bool
	DoneCount uint32
	Message   string
}

func (p *ReplyPacket) Type() uint8 { return MsgReply }

func (p *ReplyPacket) Encode(w io.Writer) error {
	status := ReplyStatusOK
	if !p.Status {
		status = ReplyStatusError
	}

	if err := binary.Write(w, binary.BigEndian, status); err != nil {
		return err
	}
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

func DecodeReplyPacket(r io.Reader) (*ReplyPacket, error) {
	var status uint8
	if err := binary.Read(r, binary.BigEndian, &status); err != nil {
		return nil, err
	}

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
		Status:    status == ReplyStatusOK,
		DoneCount: doneCount,
		Message:   string(msgBytes),
	}, nil
}

type ErrorPacket struct {
	ErrorCode uint8
	Message   string
}

func (p *ErrorPacket) Type() uint8 { return MsgError }

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
