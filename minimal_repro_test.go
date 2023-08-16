package kbucketfix

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestMinimalRepro(t *testing.T) {
	ctx, cancel := prepareTest(t, time.Duration(30*time.Second))
	defer cancel()

	host, hostDHT := makeHost(t, ctx)
	rt := hostDHT.RoutingTable()
	if rt == nil {
		t.Error()
	}
	connMgr := host.ConnManager()

	added := 0
	protected := 0
	peerAdded := rt.PeerAdded
	rt.PeerAdded = func(p peer.ID) {
		peerAdded(p)
		added += 1
		if connMgr.IsProtected(p, "") {
			protected += 1
		}
	}

	peerRemoved := rt.PeerRemoved
	rt.PeerRemoved = func(p peer.ID) {
		peerRemoved(p)
		// log.Println("PeerRemoved: " + p.String())
	}

	host.Start()

	host2, _ := makeHost(t, ctx)
	host2.Start()
	connect(host, host2, ctx)
	for i := 0; i < 3000; i += 1 {
		peerHost, _ := makeHost(t, ctx)
		peerHost.Start()
		// connect(host2, peerHost, ctx)
		connect(host, peerHost, ctx)
	}

	hostDHT.RefreshRoutingTable()

	percentage := float32(protected) / float32(added)
	const TARGET float32 = .75
	const BIAS float32 = .03
	if percentage < TARGET-BIAS || percentage > TARGET+BIAS {
		t.Error(percentage)
	}
}
