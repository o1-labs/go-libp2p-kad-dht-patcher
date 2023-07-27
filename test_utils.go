package kbucketfix

import (
	"context"
	"testing"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/require"
)

func makeHost(t *testing.T, ctx context.Context) (*bhost.BasicHost, *kaddht.IpfsDHT) {
	connMgr, _ := connmgr.NewConnManager(10, 100)
	dhtOpts := []kaddht.Option{
		kaddht.DisableAutoRefresh(),
		kaddht.Mode(kaddht.ModeServer),
	}
	hostOpt := new(bhost.HostOpts)
	hostOpt.ConnManager = connMgr
	host, err := bhost.NewHost(swarmt.GenSwarm(t, swarmt.OptDisableReuseport), hostOpt)
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
