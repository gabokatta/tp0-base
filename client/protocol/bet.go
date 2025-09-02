package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"time"
)

type Bet struct {
	FirstName string
	LastName  string
	Document  uint32
	Birthdate uint32
	Number    uint16
}

func (b *Bet) Encode(w io.Writer) error {
	if err := WriteString(w, b.FirstName); err != nil {
		return err
	}
	if err := WriteString(w, b.LastName); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, b.Document); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, b.Birthdate); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, b.Number)
}

func (b *Bet) Decode(r io.Reader) error {
	var err error
	if b.FirstName, err = ReadString(r); err != nil {
		return err
	}
	if b.LastName, err = ReadString(r); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &b.Document); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &b.Birthdate); err != nil {
		return err
	}
	return binary.Read(r, binary.BigEndian, &b.Number)
}

func NewBet(firstName, lastName, document, date, number string) (*Bet, error) {
	doc, err := strconv.ParseUint(document, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid document: %w | must be a numeric value", err)
	}

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid birthdate: %w | must have format YYYY-MM-DD", err)
	}
	birthdate := uint32(t.Year()*10000 + int(t.Month())*100 + t.Day())

	num, err := strconv.ParseUint(number, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid bet number: %w | must be a numeric value", err)
	}

	return &Bet{
		FirstName: firstName,
		LastName:  lastName,
		Document:  uint32(doc),
		Birthdate: birthdate,
		Number:    uint16(num),
	}, nil
}
