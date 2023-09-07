package common

import (
	"bufio"
	"net"
	"os"
	"strings"
	"time"

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

// Runs the client, first sending batchs of bets and then asking for the lottery result
func (c *Client) RunClient(bets string, id uint8, n int) {
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
			c.CloseClientFileDescriptor(file)
			c.CloseClientSocket(true)
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
		err := c.SendMessage(batch.BatchToBytes())
		if err != nil {
			c.LogCommunicationError("sending_batchs", err)
			c.CloseClientFileDescriptor(file)
			c.CloseClientSocket(false)
			return
		}
	}
	log.Infof("action: sending_finished_message | result: in_progress")
	c.SendMessage(NewFinished(id).FinishedToBytes())
	log.Infof("action: sending_finished_message | result: sucess")
	response, err := c.ReadMessage()

	c.CloseClientFileDescriptor(file)
	if err != nil {
		c.LogCommunicationError("sending_batchs", err)
		c.CloseClientSocket(false)
		return
	}

	batchAck := BatchAckFromBytes(response)
	if batchAck.Status == "1" {
		log.Infof("action: sending_batchs | result: success")
	} else {
		log.Infof("action: sending_batchs | result: fail")
	}
	c.GetWinners(id)
}

// Sends a Winner Packet to the server until the server responds
func (c *Client) GetWinners(id uint8) {
	finished := false
	f := 1

	for !finished {
		select {
		case <-c.sigterm_ch:
			log.Infof("action: sigterm_received")
			c.CloseClientSocket(true)
			return
		default:
		}

		winner := NewWinner(id, "1")
		c.SendMessage(winner.WinnerToBytes())

		response, err := c.ReadMessage()
		if err != nil {
			c.LogCommunicationError("consulta_ganadores", err)
			c.CloseClientSocket(false)
			return
		}
		winnerResponse := WinnerFromBytes(response)
		if winnerResponse.Status == "1" {
			log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winnerResponse.Winners))
			finished = true
			c.CloseClientSocket(false)
			break
		}
		log.Infof("action: consulta_ganadores | result: waiting")
		time.Sleep(time.Second * time.Duration(f))
		f *= 2
	}

}

// Logs a communication error
func (c *Client) LogCommunicationError(action string, err error) {
	log.Errorf("action: %v | result: fail | client_id: %v | error: %v",
		action,
		c.config.ID,
		err,
	)
}

// Closes client file descriptor and logs it
func (c *Client) CloseClientFileDescriptor(file *os.File) {
	log.Infof("action: closing_file_descriptor | result: in_progress")
	file.Close()
	log.Infof("action: closing_file_descriptor | result: success")
}

// Closes client socket and logs it
func (c *Client) CloseClientSocket(signal bool) {
	log.Infof("action: closing_socket | result: in_progress")
	c.conn.Close()
	log.Infof("action: closing_socket | result: success")
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

// Sends a message to the server, adds the necessary padding to reach blocksize.
// In case of failure returns and error
func (c *Client) SendMessage(b []byte) error {
	sentBytes := 0
	bytesToSend := len(b)
	paddingLength := BlockSize - bytesToSend
	padding := make([]byte, paddingLength)
	message := append(b, padding...)
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	for sentBytes < BlockSize {
		sent, err := c.conn.Write(message[sentBytes:])
		if err != nil {
			return err
		}

		sentBytes += sent
	}
	return nil
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
			sizeOfPacket = int(msg[3])<<8 | int(msg[2])
			sizeRead = true
		}
	}

	return msg[:sizeOfPacket], nil
}
