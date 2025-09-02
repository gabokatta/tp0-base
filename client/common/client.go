package common

import (
	"fmt"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopPeriod    time.Duration
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
// Reads the CSV line by line, building a packet of maximum 8KBs, when the limit is reached a batch is sent.
// Temporary: Since server is not concurrent -> BatchSent means socket disconnect.
func (c *Client) StartClientLoop() {
	defer c.cleanup()

	batchID := 1
	for {
		if c.signal.ShouldShutdown() {
			log.Infof("action: shutdown_requested | result: success | client_id: %v | completed_messages: %v",
				c.config.ID, batchID-1)
			break
		}

		bets, err := c.batchMaker.MakeBatch()
		if err != nil {
			log.Errorf("action: make_batch | result: fail | client_id: %v | batch_id: %v | error: %v",
				c.config.ID, batchID, err)
			break
		}

		if bets == nil || len(bets) == 0 {
			log.Infof("action: processing_complete | result: success | client_id: %v | total_batches: %v",
				c.config.ID, batchID-1)
			break
		}

		err = c.sendBetBatch(bets, batchID)
		if err != nil {
			break
		}
		batchID++

		time.Sleep(c.config.LoopPeriod)
	}

}

func (c *Client) sendBetBatch(bets []protocol.Bet, batchID int) error {
	log.Debugf("action: send_batch | result: in_progress | client_id: %v | batch_id: %v | bet_count: %v",
		c.config.ID, batchID, len(bets))

	response, err := c.network.SendBetBatch(c.config.ID, bets)
	if err != nil {
		log.Errorf("action: send_batch | result: fail | client_id: %v | batch_id: %v | bet_count: %v | error: %v",
			c.config.ID, batchID, len(bets), err)
		return err
	}

	c.handleResponse(response, bets, batchID)

	return nil
}

// Given the server response, this function takes into account the different packet types in order to act.
func (c *Client) handleResponse(response protocol.Packet, bets []protocol.Bet, iteration int) {
	switch resp := response.(type) {
	case *protocol.ReplyPacket:
		log.Infof("action: apuestas_enviada | result: success | batch_size: %v | batch_id: %v", len(bets), iteration)
	case *protocol.ErrorPacket:
		log.Errorf("action: apuestas_enviada | result: fail | client_id: %v | batch_size: %v | batch_id: %v | error_code: %v | msg: %v",
			c.config.ID, len(bets), iteration, protocol.ErrorFromPacket(*resp), resp.Message)
	default:
		log.Errorf("action: apuestas_enviada | result: fail | client_id: %v | batch_id: %v | error: unknown_response_type", c.config.ID, iteration)
	}
}

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
