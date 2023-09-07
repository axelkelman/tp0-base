package common

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

const BlockSize = 128

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
}

// Client Entity that encapsulates how
type Client struct {
	config     ClientConfig
	conn       net.Conn
	sigterm_ch chan os.Signal
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, sigterm_ch chan os.Signal) *Client {
	client := &Client{
		config:     config,
		sigterm_ch: sigterm_ch,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) SendClientBet(b Bet) {

	select {
	case <-c.sigterm_ch:
		log.Infof("action: sigterm_received")
		return
	default:
	}

	log.Debugf("action: showing_bet | result: success | name: %v | surname: %v | document: %v | birthday: %v | number: %v",
		b.Data.Name,
		b.Data.Surname,
		b.Data.Document,
		b.Data.Birthday,
		b.Data.Number,
	)

	// Create the connection to the server
	c.createClientSocket()

	bet := b.BetToBytes()

	c.SendMessage(bet)

	msg, err := c.ReadMessage()

	c.conn.Close()
	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	response := BetAckFromBytes(msg)

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
		response.Data.Document,
		response.Data.Number,
	)

}

// Sends a message to the server
func (c *Client) SendMessage(b []byte) {
	sentBytes := 0
	bytesToSend := len(b)
	paddingLength := BlockSize - bytesToSend
	padding := make([]byte, paddingLength)
	message := append(b, padding...)
	for sentBytes < BlockSize {
		sent, err := c.conn.Write(message[sentBytes:])

		if err != nil {
			return
		}

		sentBytes += sent
	}

}

// Receives a message from the server. In
// case of failure returns and error
func (c *Client) ReadMessage() (bytes []byte, err error) {
	msg := make([]byte, BlockSize)
	readBytes := 0
	sizeOfPacket := 1
	size_read := false //Indicates whether the size of the packet has already been read or not
	for readBytes < BlockSize {
		read, err := c.conn.Read(msg[readBytes:])
		if err != nil {
			return msg, err
		}
		readBytes += read
		if !size_read {
			sizeOfPacket = int(msg[2])
			size_read = true
		}
	}

	return msg[:sizeOfPacket], nil
}
