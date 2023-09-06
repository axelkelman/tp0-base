package common

import (
	"strings"
)

const BetPacketId = 1
const BatchPacketId = 3
const FinishedPacketId = 4

// Packet Header common to every packet in the protocol
type PacketHeader struct {
	PacketType uint8
	ID         uint8
}

// Converts PacketHeader to bytes
func (h *PacketHeader) HeaderToBytes(PayloadSize int) []byte {
	bytes := []byte{h.PacketType, h.ID}
	size := uint16(PayloadSize + 4)
	sizeBytes := make([]byte, 2)
	sizeBytes[0] = byte(size)
	sizeBytes[1] = byte(size >> 8)
	bytes = append(bytes, sizeBytes...)
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
			PacketType: BetPacketId,
			ID:         id,
		},
		Data: data,
	}
	return bet
}

// Converts a Bet struct into an array of bytes
func (b *Bet) BetToBytes() []byte {
	formatedString := b.Data.Name + "|" + b.Data.Surname + "|" + b.Data.Document + "|" + b.Data.Birthday + "|" + b.Data.Number
	headerBytes := b.Header.HeaderToBytes((len(formatedString)))
	payloadBytes := []byte(formatedString)
	return append(headerBytes, payloadBytes...)
}

// Batch packet
type Batch struct {
	Header PacketHeader
	Bets   []Bet
}

// Returns a new batch packet containing bets
func NewBatch(bets []string, id uint8) *Batch {
	header := PacketHeader{
		PacketType: BatchPacketId,
		ID:         id,
	}

	var batchBets []Bet
	for i := 0; i < len(bets); i++ {
		fields := strings.Split(bets[i], ",")
		betData := BetData{
			Name:     fields[0],
			Surname:  fields[1],
			Document: fields[2],
			Birthday: fields[3],
			Number:   fields[4],
		}
		bet := NewBet(betData, id)
		batchBets = append(batchBets, *bet)
	}

	batch := &Batch{
		Header: header,
		Bets:   batchBets,
	}

	return batch

}

// Converts a Batch packet into an array of bytes
func (b *Batch) BatchToBytes() []byte {
	betNumber := uint8(len(b.Bets))
	payloadBytes := []byte{betNumber}
	for i := 0; i < len(b.Bets); i++ {
		payloadBytes = append(payloadBytes, b.Bets[i].BetToBytes()...)
	}
	headerBytes := b.Header.HeaderToBytes(len(payloadBytes))
	return append(headerBytes, payloadBytes...)
}

// Finished packet
type Finished struct {
	Header PacketHeader
}

// Returns a new finished packet
func NewFinished(id uint8) *Finished {
	finished := &Finished{
		Header: PacketHeader{
			PacketType: FinishedPacketId,
			ID:         id,
		},
	}
	return finished
}

// Converts a finished packet into an array of bytes
func (f *Finished) FinishedToBytes() []byte {
	return f.Header.HeaderToBytes(0)
}

// BatchAck packet
type BatchAck struct {
	Header PacketHeader
	Status string
}

// Providing an array of bytes, returns a BatchAck packet
func BatchAckFromBytes(bytes []byte) *BatchAck {
	status := string(bytes[4])
	ack := &BatchAck{
		Header: PacketHeader{
			PacketType: bytes[0],
			ID:         bytes[1],
		},
		Status: status,
	}
	return ack
}
