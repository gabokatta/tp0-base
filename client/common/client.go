package common

import (
	"errors"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type ClientConfig struct {
	ID            string
	ServerAddress string
	BatchConfig   BatchConfig
}

// Client responsible for managing outgoing connections to the server.
// Uses a signalHandler to be aware of graceful shutdowns.
// The Network struct allows it to safely send messages to the server.
type Client struct {
	config     ClientConfig
	signal     *SignalHandler
	network    *protocol.Network
	batchMaker *BatchMaker
}

// NewClient Returns a new Client based on batch and client configs.
// Using the server address stored in the config it instantiates a network struct.
// The signalHandler is registered when this function is called.
// A BatchMaker is created for bet batch processing.
func NewClient(clientConfig ClientConfig, batchConfig BatchConfig) (*Client, error) {

	signal := NewSignalHandler()
	network := protocol.NewNetwork(clientConfig.ServerAddress)
	batchMaker, err := NewBatchMaker(
		clientConfig.ID,
		batchConfig,
		signal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch processor: %w", err)
	}

	return &Client{
		clientConfig,
		signal,
		network,
		batchMaker,
	}, nil
}

// StartClientLoop Handles the main logic-loop of the client application
// Reads the CSV line by line, building a packet of n maximumBytes, when the limit is reached a batch is sent.
// Temporary: Since server is not concurrent -> BatchSent means socket disconnect.
func (c *Client) StartClientLoop() {
	defer c.cleanup()

	if err := c.sendBetStart(); err != nil {
		return
	}

	if err := c.sendBets(); err != nil {
		return
	}

	if err := c.sendBetFinish(); err != nil {
		return
	}

}

// Sends a bet batch to the server and awaits the confirmation.
func (c *Client) sendBetStart() error {
	log.Infof("action: send_batch_start | result: in_progress | client_id: %v",
		c.config.ID)

	response, err := c.network.SendStartBet(c.config.ID)
	if err != nil {
		log.Errorf("action: send_batch_start | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return err
	}

	return c.handleStartResponse(response)
}

// sendBets loops through all the possible batches and sends them to the server.
func (c *Client) sendBets() error {
	batchID := 1
	for {
		if c.signal.ShouldShutdown() {
			log.Infof("action: shutdown_requested | result: success | client_id: %v | completed_messages: %v",
				c.config.ID, batchID-1)
			return errors.New("client got a shutdown signal")
		}

		bets, err := c.batchMaker.MakeBatch()
		if err != nil {
			log.Errorf("action: make_batch | result: fail | client_id: %v | batch_id: %v | error: %v",
				c.config.ID, batchID, err)
			return err
		}

		if bets == nil || len(bets) == 0 {
			log.Infof("action: processing_complete | result: success | client_id: %v | total_batches: %v",
				c.config.ID, batchID-1)
			break
		}

		err = c.sendSingleBatch(bets, batchID)
		if err != nil {
			return err
		}
		batchID++
	}
	return nil
}

// Sends a bet batch to the server and awaits the confirmation.
func (c *Client) sendSingleBatch(bets []protocol.Bet, batchID int) error {
	log.Infof("action: send_batch | result: in_progress | client_id: %v | batch_id: %v | bet_count: %v",
		c.config.ID, batchID, len(bets))

	response, err := c.network.SendBetBatch(c.config.ID, bets)
	if err != nil {
		log.Errorf("action: send_batch | result: fail | client_id: %v | batch_id: %v | bet_count: %v | error: %v",
			c.config.ID, batchID, len(bets), err)
		return err
	}

	return c.handleBetResponse(response, bets, batchID)
}

// Sends the BetFinishPacket to the server and awaits confirmation.
func (c *Client) sendBetFinish() error {
	log.Debugf("action: send_bet_finish | result: in_progress | client_id: %v", c.config.ID)

	res, err := c.network.SendFinishBet(c.config.ID)
	if err != nil {
		log.Errorf("action: send_bet_finish | result: fail | error: %v", err)
		return err
	}

	return c.handleFinishResponse(res)
}

// Given the server response to the finish, this function takes into account the different packet types in order to act.
func (c *Client) handleStartResponse(response protocol.Packet) error {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: send_batch_start | result: success")
		return nil
	case *protocol.ErrorPacket:
		log.Infof("action: send_batch_start | result: error | code: %v | msg: %v", protocol.ErrorFromPacket(*resp), resp.Message)
		return errors.New("server could not successfully acknowledge the BetStartPacket")
	default:
		log.Errorf("action: send_batch_start | result: fail | client_id: %v | error: unknown_response_type", c.config.ID)
		return errors.New("server responded with an invalid error type")
	}
}

// Given the server response to the bets, this function takes into account the different packet types in order to act.
func (c *Client) handleBetResponse(response protocol.Packet, bets []protocol.Bet, iteration int) error {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: apuestas_enviada | result: success | batch_size: %v | batch_id: %v", len(bets), iteration)
		return nil
	case *protocol.ErrorPacket:
		log.Errorf("action: apuestas_enviada | result: fail | client_id: %v | batch_size: %v | batch_id: %v | error_code: %v | msg: %v",
			c.config.ID, len(bets), iteration, protocol.ErrorFromPacket(*resp), resp.Message)
		return fmt.Errorf("server failed to store bet batch with ID: %v", iteration)
	default:
		log.Errorf("action: apuestas_enviada | result: fail | client_id: %v | batch_id: %v | error: unknown_response_type", c.config.ID, iteration)
		return errors.New("server responded with an invalid error type")
	}
}

// Given the server response to the finish, this function takes into account the different packet types in order to act.
func (c *Client) handleFinishResponse(response protocol.Packet) error {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: finish_ack | result: success")
		return nil
	case *protocol.ErrorPacket:
		log.Infof("action: finish_ack | result: error | code: %v | msg: %v", protocol.ErrorFromPacket(*resp), resp.Message)
		return errors.New("server could not successfully acknowledge the BetFinishPacket")
	default:
		log.Errorf("action: finish_ack | result: fail | client_id: %v | error: unknown_response_type", c.config.ID)
		return errors.New("server responded with an invalid error type")
	}
}

// closes up all open file descriptors.
func (c *Client) cleanup() {
	log.Infof("action: clean_up | result: in_progress | client_id: %v", c.config.ID)

	if c.batchMaker != nil {
		if err := c.batchMaker.Close(); err != nil {
			log.Warningf("action: batchmaker_close | result: fail | client_id: %v | error: %v",
				c.config.ID, err)
		} else {
			log.Infof("action: closing_bet_file | status: success")
		}
	}

	if c.network != nil {
		if err := c.network.Disconnect(); err != nil {
			log.Warningf("action: network_disconnect | result: fail | client_id: %v | error: %v",
				c.config.ID, err)
		} else {
			log.Infof("action: closing_network | status: success")
		}

	}

	if c.signal != nil {
		c.signal.Cleanup()
		log.Infof("action: closing_signal_channel | status: success")
	}

	log.Infof("action: clean_up | result: success | client_id: %v", c.config.ID)
}
