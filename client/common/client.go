package common

import (
	"bufio"
	"net"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const BlockSize = 8192

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

// SendClientBets sends all client bets in batch of size n max
func (c *Client) SendClientBets(bets string, id uint8, n int) {
	file, err := os.Open(c.GetBetsPath(bets))
	if err != nil {
		log.Errorf("action: opening_bets_file | result: fail | error: %v", err)
		return
	}
	c.createClientSocket()
	scanner := bufio.NewScanner(file)
	log.Infof("action: sending_batchs | result: in_progress")
loop:
	for {
		select {
		case <-c.sigterm_ch:
			log.Infof("action: sigterm_received")
			c.Shutdown(file, true)
			return
		default:
		}

		var batchBets []string
		for i := 0; i < n; i++ {
			if !scanner.Scan() {
				if i == 0 {
					break loop
				}
				break
			}
			batchBets = append(batchBets, scanner.Text())
		}

		batch := NewBatch(batchBets, id)
		batchBytes := batch.BatchToBytes()
		c.SendMessage(batchBytes)
	}
	log.Infof("action: sending_finished_message | result: in_progress")
	c.SendMessage(NewFinished(id).FinishedToBytes())
	log.Infof("action: sending_finished_message | result: sucess")
	response, err := c.ReadMessage()

	c.Shutdown(file, false)
	if err != nil {
		log.Errorf("action: sending_batchs | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	batchAck := BatchAckFromBytes(response)
	if batchAck.Status == "1" {
		log.Infof("action: sending_batchs | result: success")
	} else {
		log.Infof("action: sending_batchs | result: fail")
	}

}

// Closes client socket and file descriptor
func (c *Client) Shutdown(file *os.File, signal bool) {
	log.Infof("action: closing_socket | result: in_progress")
	c.conn.Close()
	log.Infof("action: closing_socket | result: success")
	log.Infof("action: closing_file_descriptor | result: in_progress")
	file.Close()
	log.Infof("action: closing_file_descriptor | result: success")
	if signal {
		log.Infof("action: exiting_gracefully | result: success")
	}
}

// Returns the path of the bets csv file which corresponds to this client
func (c *Client) GetBetsPath(bets string) string {
	aux := strings.Split(bets, "-")
	path := aux[0] + "-" + c.config.ID + ".csv"
	log.Infof("csv_path_file : %v", path)
	return path
}

// Sends a message to the server, adds the necessary padding to reach blocksize
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
	sizeOfPacket := 0
	sizeRead := false //Indicates whether the size of the packet has already been read or not
	for readBytes < BlockSize {
		read, err := c.conn.Read(msg[readBytes:])
		if err != nil {
			return msg, err
		}
		readBytes += read
		if !sizeRead {
			sizeOfPacket = int(msg[2])
			sizeRead = true
		}
	}

	return msg[:sizeOfPacket], nil
}
