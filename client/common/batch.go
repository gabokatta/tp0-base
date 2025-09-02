package common

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/protocol"
	"io"
	"os"
	"strings"
)

type BatchConfig struct {
	MaxBytes  uint32
	MaxAmount uint32
}

type BatchMaker struct {
	clientID   string
	csv        *os.File
	reader     *csv.Reader
	signal     *SignalHandler
	config     BatchConfig
	reachedEOF bool
}

func getBetsLocation(clientID string) string {
	return fmt.Sprintf("./.data/agency-%s.csv", clientID)
}

func NewBatchMaker(clientID string, config BatchConfig, signal *SignalHandler) (*BatchMaker, error) {
	path := getBetsLocation(clientID)
	log.Debugf("action: load_csv | result: in_progress | file: %s", path)
	file, err := os.Open(path)
	if err != nil {
		log.Errorf("action: load_csv | result: fail | file: %s", path)
		return nil, fmt.Errorf("failed to open CSV: %w", err)
	}

	log.Infof("action: load_csv | result: success | file: %s", file.Name())

	reader := csv.NewReader(bufio.NewReader(file))
	reader.FieldsPerRecord = 5
	reader.TrimLeadingSpace = true

	return &BatchMaker{
		clientID:   clientID,
		csv:        file,
		reader:     reader,
		signal:     signal,
		config:     config,
		reachedEOF: false,
	}, nil
}

// MakeBatch returns a collection of bets that fit within the config.yaml limits.
// It returns nil if we reached a shutdown or the end of file.
func (bm *BatchMaker) MakeBatch() ([]protocol.Bet, error) {
	if bm.reachedEOF || bm.signal.ShouldShutdown() {
		return nil, nil
	}

	var bets []protocol.Bet
	for {
		if bm.shouldStopBatch(bets) {
			break
		}

		bet, err := bm.readNextBet()
		if err != nil {
			return nil, err
		}
		if bm.reachedEOF {
			break
		}

		if bm.canAddToBatch(bets, *bet) {
			bets = append(bets, *bet)
		} else {
			break
		}
	}

	if len(bets) > 0 {
		log.Debugf("action: batch_created | result: success | client_id: %v | bet_count: %v",
			bm.clientID, len(bets))
	}

	return bets, nil
}

func (bm *BatchMaker) readNextBet() (*protocol.Bet, error) {
	record, err := bm.reader.Read()
	if err == io.EOF {
		bm.reachedEOF = true
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV record: %w", err)
	}

	bet, err := bm.parseCSVRecord(record)
	if err != nil {
		return nil, fmt.Errorf("invalid CSV record found - stopping client: %w | record: %v", err, record)
	}

	return bet, nil
}

func (bm *BatchMaker) parseCSVRecord(record []string) (*protocol.Bet, error) {
	if len(record) != 5 {
		return nil, fmt.Errorf("invalid CSV record: expected exactly 5 fields, got %d", len(record))
	}

	for i := range record {
		record[i] = strings.TrimSpace(record[i])
	}
	return protocol.NewBet(record[0], record[1], record[2], record[3], record[4])
}

func (bm *BatchMaker) canAddToBatch(currentBets []protocol.Bet, newBet protocol.Bet) bool {
	if len(currentBets) == 0 {
		return true
	}

	tempBets := append(currentBets, newBet)
	estimatedSize, err := bm.estimateBatchSize(tempBets)
	if err != nil {
		log.Warningf("action: estimate_batch_size | result: fail | client_id: %v | error: %v",
			bm.clientID, err)
		return false
	}

	return estimatedSize <= bm.config.MaxBytes
}

// estimateBatchSize mocks packet creation to see if current bets will fit the total byte limit.
func (bm *BatchMaker) estimateBatchSize(bets []protocol.Bet) (uint32, error) {
	packet, err := protocol.NewBetPacket(bm.clientID, bets)
	if err != nil {
		return 0, err
	}

	var buf bytes.Buffer
	if err := packet.Encode(&buf); err != nil {
		return 0, err
	}

	return uint32(buf.Len()) + protocol.HeaderSize, nil
}

func (bm *BatchMaker) shouldStopBatch(bets []protocol.Bet) bool {
	if bm.signal.ShouldShutdown() {
		log.Infof("action: batch_processing | result: interrupted | client_id: %v | processed_bets: %v",
			bm.clientID, len(bets))
		return true
	}

	if uint32(len(bets)) >= bm.config.MaxAmount {
		return true
	}

	return false
}

func (bm *BatchMaker) Close() error {
	log.Debugf("action: closing_bet_file | status: in_progress")
	if bm.csv != nil {
		if err := bm.csv.Close(); err != nil {
			return fmt.Errorf("failed to close CSV file: %w", err)
		}
		bm.csv = nil
	}
	return nil
}
