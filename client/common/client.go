package common

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

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
	sent_bytes := 0
	bytes_to_send := len(b)
	for sent_bytes < bytes_to_send {
		sent, err := c.conn.Write(b[sent_bytes:])

		if err != nil {
			return
		}

		sent_bytes += sent
	}

}

// Receives a message from the server. In
// case of failure returns and error
func (c *Client) ReadMessage() (bytes []byte, err error) {
	msg := make([]byte, 1024)
	read_bytes := 0
	size_of_packet := 1
	size_read := false //Indicates whether the size of the packet has already been read or not
	for read_bytes < size_of_packet {
		read, err := c.conn.Read(msg[read_bytes:])
		if err != nil {
			return msg, err
		}
		read_bytes += read
		if !size_read {
			size_of_packet = int(msg[2])
			size_read = true
		}
	}

	return msg, nil
}
