package common

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type SignalHandler struct {
	channel chan os.Signal
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewSignalHandler() *SignalHandler {
	ctx, cancel := context.WithCancel(context.Background())
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGTERM, syscall.SIGINT)

	sh := &SignalHandler{
		channel: channel,
		ctx:     ctx,
		cancel:  cancel,
	}

	go sh.listen()

	return sh
}

func (sh *SignalHandler) listen() {
	sig := <-sh.channel
	log.Warningf("action: signal_received | result: in_progress | code: %v", sig)
	sh.cancel()
}

func (sh *SignalHandler) shouldShutdown() bool {
	select {
	case <-sh.ctx.Done():
		return true
	default:
		return false
	}
}

func (sh *SignalHandler) Cleanup() {
	log.Infof("action: closing_signal_channel | status: in_progress")
	signal.Stop(sh.channel)
	close(sh.channel)
	log.Infof("action: closing_signal_channel | status: success")
}
