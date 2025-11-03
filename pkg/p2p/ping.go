package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Our custom protocol ID
const PingProtocolID = "/gotangle/ping/1.0.0"

// StartPingHandler registers a handler to respond to ping messages.
func StartPingHandler(h host.Host) {
	h.SetStreamHandler(PingProtocolID, func(s network.Stream) {
		defer s.Close()
		r := bufio.NewReader(s)
		msg, err := r.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading ping: %v\n", err)
			return
		}
		fmt.Printf("📩 Received: %s", msg)
		_, _ = s.Write([]byte("pong\n"))
	})
}

// SendPing sends a ping message to a given peer.
func SendPing(ctx context.Context, h host.Host, pid peer.ID) error {
	s, err := h.NewStream(ctx, pid, PingProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	defer s.Close()

	start := time.Now()
	_, _ = s.Write([]byte("ping\n"))

	resp, err := bufio.NewReader(s).ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read response error: %v", err)
	}

	fmt.Printf("📤 Sent ping to %s → got %s (rtt=%v)\n", pid, resp, time.Since(start))
	return nil
}
