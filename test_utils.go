package kbucketfix

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/require"
)

func prepareTest(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-time.After(timeout):
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			fmt.Printf("%s", buf)
			cancel()
			t.Errorf("timeout exceeded")
			// panic("timeout exceeded")
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func makeHost(t *testing.T, ctx context.Context) (*bhost.BasicHost, *kaddht.IpfsDHT) {
	connMgr, _ := connmgr.NewConnManager(10, 100)
	dhtOpts := []kaddht.Option{
		kaddht.DisableAutoRefresh(),
		kaddht.Mode(kaddht.ModeServer),
	}
	hostOpt := new(bhost.HostOpts)
	hostOpt.ConnManager = connMgr
	host, err := bhost.NewHost(swarmt.GenSwarm(t, swarmt.OptDisableReuseport, swarmt.OptDisableQUIC), hostOpt)
	require.NoError(t, err)
	hostDHT, err := kaddht.New(ctx, host, dhtOpts...)
	require.NoError(t, err)
	return host, hostDHT
}

func connect(a, b *bhost.BasicHost, ctx context.Context) {
	hi := peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()}
	a.Peerstore().AddAddrs(hi.ID, hi.Addrs, peerstore.PermanentAddrTTL)
	a.Connect(ctx, hi)
}
