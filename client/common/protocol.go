package common

import (
	"strings"
)

// Packet Header common to every packet in the protocol
type PacketHeader struct {
	PacketType uint8
	ID         uint8
}

// Converts PacketHeader to bytes
func (h *PacketHeader) HeaderToBytes(PayloadSize uint8) []byte {
	bytes := []byte{h.PacketType, h.ID, PayloadSize + 3}
	return bytes
}

// Data from a bet
type BetData struct {
	Name     string
	Surname  string
	Document string
	Birthday string
	Number   string
}

// Bet packet
type Bet struct {
	Header PacketHeader
	Data   BetData
}

// Returns a new Bet Packet
func NewBet(data BetData, id uint8) *Bet {
	bet := &Bet{
		Header: PacketHeader{
			PacketType: 1,
			ID:         id,
		},
		Data: data,
	}
	return bet
}

// Converts a Bet struct into an array of bytes
func (b *Bet) BetToBytes() []byte {
	formated_string := b.Data.Name + "|" + b.Data.Surname + "|" + b.Data.Document + "|" + b.Data.Birthday + "|" + b.Data.Number
	header_bytes := b.Header.HeaderToBytes(uint8(len(formated_string)))
	payload_bytes := []byte(formated_string)
	return append(header_bytes, payload_bytes...)
}

// Data from a BetAck Packet sent from the server
type BetAckData struct {
	Document string
	Number   string
	Status   string
}

// BetAck packet
type BetAck struct {
	Header PacketHeader
	Data   BetAckData
}

// Converts an array of bytes into a BetAckPacket
func BetAckFromBytes(bytes []byte) *BetAck {
	data := string(bytes[3:])
	fields := strings.Split(data, "|")
	betAck := &BetAck{
		Header: PacketHeader{
			PacketType: bytes[0],
			ID:         bytes[1],
		},
		Data: BetAckData{
			Document: fields[0],
			Number:   fields[1],
			Status:   fields[2],
		},
	}
	return betAck
}
