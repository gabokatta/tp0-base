package common

import (
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client responsible for managing outgoing connections to the server.
// Uses a signalHandler to be aware of graceful shutdowns.
// The Network struct allows it to safely send messages to the server.
type Client struct {
	config  ClientConfig
	signal  *SignalHandler
	network *protocol.Network
	bet     *protocol.Bet
}

// NewClient Returns a new Client given the config.
// Using the server address stored in the config it instantiates a network struct.
// The signalHandler is registered when this function is called.
func NewClient(config ClientConfig) *Client {
	return &Client{
		config:  config,
		signal:  NewSignalHandler(),
		network: protocol.NewNetwork(config.ServerAddress),
	}
}

// Initialize loads the bet from the environment variables (temporary)
func (c *Client) Initialize() error {
	bet, err := generateBetFromEnv()
	if err != nil {
		log.Errorf("action: initialize | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return err
	}
	c.bet = bet

	log.Infof("action: initialize | result: success | client_id: %v | dni: %v | numero: %v",
		c.config.ID, bet.Document, bet.Number)
	return nil
}

// StartClientLoop Handles the main logic-loop of the client application
// After initializing the bet to send, the loop begins and on each iteration the client checks if it needs to shutdown.
// On each iteration the client sleeps and when its done we exit de loop using the clean up deferred function to clean
// all resources.
func (c *Client) StartClientLoop() {
	defer c.cleanup()

	if err := c.Initialize(); err != nil {
		return
	}

	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		if c.signal.ShouldShutdown() {
			log.Infof("action: shutdown_requested | result: success | client_id: %v | completed_messages: %v",
				c.config.ID, msgID-1)
			return
		}
		c.sendBet(msgID)
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v | total_messages: %v",
		c.config.ID, c.config.LoopAmount)
}

// This function uses the network to send the packet and logs the error if they exist.
// After sending the server sends a response and the client handles it.
func (c *Client) sendBet(iteration int) {
	log.Debugf("action: send_bet | result: in_progress | client_id: %v | iteration: %v", c.config.ID, iteration)
	response, err := c.network.SendBet(c.config.ID, *c.bet)
	if err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | iteration: %v | dni: %v | error: %v",
			c.config.ID, iteration, c.bet.Document, err)
		return
	}

	c.handleResponse(response, *c.bet, iteration)
}

// Given the server response, this function takes into account the different packet types in order to act.
func (c *Client) handleResponse(response protocol.Packet, bet protocol.Bet, iteration int) {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.Document, bet.Number)
	case *protocol.ErrorPacket:
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | iteration: %v | dni: %v | numero: %v | error_code: %v | msg: %v",
			c.config.ID, iteration, bet.Document, bet.Number, protocol.ErrorFromPacket(*resp), resp.Message)
	default:
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | iteration: %v | error: unknown_response_type", c.config.ID, bet.Number)
	}
}

func (c *Client) cleanup() {
	if c.signal != nil {
		c.signal.Cleanup()
	}

	if c.network != nil {
		if err := c.network.Disconnect(); err != nil {
			log.Warningf("action: network_disconnect | result: fail | client_id: %v | error: %v",
				c.config.ID, err)
		}
	}

	log.Infof("action: clean_up | result: success | client_id: %v", c.config.ID)
}

// Funcion temporal, entiendo que luego vamos a leer desde archivos.
func generateBetFromEnv() (*protocol.Bet, error) {
	requiredVars := []string{"NOMBRE", "APELLIDO", "DOCUMENTO", "NACIMIENTO", "NUMERO"}
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			log.Errorf("action: env_validation | result: fail | missing_var: %v", varName)
		}
	}

	bet, err := protocol.NewBet(
		os.Getenv("NOMBRE"),
		os.Getenv("APELLIDO"),
		os.Getenv("DOCUMENTO"),
		os.Getenv("NACIMIENTO"),
		os.Getenv("NUMERO"),
	)

	if err != nil {
		log.Errorf("action: bet_creation | result: fail | error: %v", err)
		return nil, err
	}

	return bet, nil
}
