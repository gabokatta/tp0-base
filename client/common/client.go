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

type Client struct {
	config  ClientConfig
	signal  *SignalHandler
	network *protocol.Network
	bet     *protocol.Bet
}

func NewClient(config ClientConfig) *Client {
	return &Client{
		config:  config,
		signal:  NewSignalHandler(),
		network: protocol.NewNetwork(config.ServerAddress),
	}
}

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

func (c *Client) handleResponse(response protocol.Packet, bet protocol.Bet, iteration int) {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: apuesta_enviada | result: success | client_id: %v | iteration: %v | dni: %v | numero: %v",
			c.config.ID, iteration, bet.Document, bet.Number)
	case *protocol.ErrorPacket:
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | iteration: %v | dni: %v | numero: %v | error_code: %v | msg: %v",
			c.config.ID, iteration, bet.Document, bet.Number, protocol.ErrorFromPacket(*resp), resp.Message)
	default:
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | iteration: %v | dni: %v | numero: %v | error: unknown_response_type_%v",
			c.config.ID, iteration, bet.Document, bet.Number, response.Type())
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
