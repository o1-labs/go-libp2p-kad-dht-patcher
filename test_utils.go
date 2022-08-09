package kbucketfix

import (
	"context"
	"testing"
	"time"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/stretchr/testify/require"
)

func makeHost(t *testing.T, ctx context.Context) (*bhost.BasicHost, *kaddht.IpfsDHT) {
	connMgr := connmgr.NewConnManager(10, 100, 10*time.Second)
	dhtOpts := []kaddht.Option{
		kaddht.DisableAutoRefresh(),
		kaddht.Mode(kaddht.ModeServer),
	}
	hostOpt := new(bhost.HostOpts)
	hostOpt.ConnManager = connMgr
	host, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx, swarmt.OptDisableReuseport), hostOpt)
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
