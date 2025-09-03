package shutdown

import (
	"context"
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"syscall"
)

var log = logging.MustGetLogger("log")

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
	log.Debugf("action: signal_received | result: success | code: %v", sig)
	sh.cancel()
}

func (sh *SignalHandler) ShouldShutdown() bool {
	select {
	case <-sh.ctx.Done():
		return true
	default:
		return false
	}
}

func (sh *SignalHandler) Cleanup() {
	log.Debugf("action: closing_signal_channel | status: in_progress")
	signal.Stop(sh.channel)
	close(sh.channel)
}
