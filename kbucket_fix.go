package kbucketfix

import (
	"sync"
	"time"

	"github.com/elliotchance/orderedmap"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	kbucketTag       = "kbucket"
	protectedBuckets = 2
	// BaseConnMgrScore is the base of the score set on the connection
	// manager "kbucket" tag. It is added with the common prefix length
	// between two peer IDs.
	baseConnMgrScore = 5
)

type DHTPeerProtectionPatcher struct {
	// Max number of peers to protect
	// non-positive means unlimited
	// default is 0
	MaxProtected int
	// Target percentage of protected peers, (0.0,1.0]
	// default is 0.5
	ProtectionRate float32

	lock sync.RWMutex
	// OrderedMap it an associative array that preserves key insertion order
	// which serves a different purpose from SortedMap or PriorityQueue
	// The performance of OrderedMap is not too worse than map + container/list solution
	// so keep using it for now to keep the code simple
	dist2protected map[int]*orderedmap.OrderedMap // OrderedMap types: map[peer.ID]time.Time
	dist2tagged    map[int]*orderedmap.OrderedMap // OrderedMap types: map[peer.ID]time.Time

	dht          *kaddht.IpfsDHT
	host         host.Host
	connMgr      connmgr.ConnManager
	selfId       kb.ID
	routingTable *kb.RoutingTable
}

func (p *DHTPeerProtectionPatcher) getProtectedLenThreadUnsafe() int {
	length := 0
	for _, m := range p.dist2protected {
		length += m.Len()
	}
	return length
}

func (p *DHTPeerProtectionPatcher) getTaggedLenThreadUnsafe() int {
	length := 0
	for _, m := range p.dist2tagged {
		length += m.Len()
	}
	return length
}

func (p *DHTPeerProtectionPatcher) isMaxProtectedReachedThreadUnsafe() bool {
	if p.MaxProtected <= 0 {
		return false
	}
	return p.getProtectedLenThreadUnsafe() >= p.MaxProtected
}

// func (p *DHTPeerProtectionPatcher) getProtectionRate() float32 {
// 	p.lock.RLock()
// 	defer p.lock.RUnlock()
// 	return p.getProtectionRateThreadUnsafe()
// }

func (p *DHTPeerProtectionPatcher) getProtectionRateThreadUnsafe() float32 {
	protectedLen := p.getProtectedLenThreadUnsafe()
	taggedLen := p.getTaggedLenThreadUnsafe()
	return float32(protectedLen) / float32(protectedLen+taggedLen)
}

func (p *DHTPeerProtectionPatcher) adjustProtectedThreadUnsafe() {
	maxReached := p.isMaxProtectedReachedThreadUnsafe()
	nActions := 0
	if maxReached {
		// swap at most 1 when a new peer is added
		nActions = 1
	} else if p.getProtectionRateThreadUnsafe() < p.ProtectionRate {
		// Calculate the number of peers that need to be moved from tagged to protected
		protected := p.getProtectedLenThreadUnsafe()
		total := protected + p.getTaggedLenThreadUnsafe()
		targetProtected := int(float32(total) * p.ProtectionRate)
		if p.MaxProtected > 0 && p.MaxProtected < targetProtected {
			targetProtected = p.MaxProtected
		}
		nActions = targetProtected - protected
	} else {
		// Do nothing when protection rate is above threshold
		// It's not likely needed to prune protected peers in this case,
		// Remember to uncomment p.adjustProtectedThreadUnsafe() in PeerRemoved callback when
		// the prune logic is in place
		return
	}
	// Only make the adjustment when protection rate is lower than threshold
	for i := 0; i < nActions; i++ {
		minDistTagged := -1
		for d, m := range p.dist2tagged {
			if m.Len() > 0 {
				if minDistTagged < 0 || d < minDistTagged {
					minDistTagged = d
				}
			}
		}
		if minDistTagged < 0 {
			return
		}
		maxDistProtected := -1
		for d, m := range p.dist2protected {
			if m.Len() > 0 {
				if maxDistProtected < 0 || d > maxDistProtected {
					maxDistProtected = d
				}
			}
		}

		taggedBucket := p.dist2tagged[minDistTagged]
		bestTagged := taggedBucket.Back()
		bestTaggedPeerId := bestTagged.Key.(peer.ID)
		bestTaggedTime := bestTagged.Value.(time.Time)

		// When max value is set and reached
		// we need to perform a swap here
		if maxReached {
			// Or maybe we can replace oldest protected peer with latest tagged peer
			// When distances are the same
			if minDistTagged >= maxDistProtected {
				return
			}

			protectedBucket := p.dist2protected[maxDistProtected]
			worstProtected := protectedBucket.Front()
			worstProtectedPeerId := worstProtected.Key.(peer.ID)
			worstProtectedTime := worstProtected.Value.(time.Time)
			// Swap
			taggedBucket.Delete(bestTagged.Key)
			protectedBucket.Delete(worstProtected.Key)
			insertThreadUnsafe(p.dist2tagged, maxDistProtected, worstProtectedPeerId, worstProtectedTime)
			insertThreadUnsafe(p.dist2protected, minDistTagged, bestTaggedPeerId, bestTaggedTime)
			p.connMgr.Unprotect(worstProtectedPeerId, kbucketTag)
			p.connMgr.TagPeer(worstProtectedPeerId, kbucketTag, baseConnMgrScore)
			p.connMgr.Protect(bestTaggedPeerId, kbucketTag)
		} else {
			// Otherwise just move the selected peer from tagged bucket to protected bucket
			taggedBucket.Delete(bestTagged.Key)
			insertThreadUnsafe(p.dist2protected, minDistTagged, bestTaggedPeerId, bestTaggedTime)
			p.connMgr.Protect(bestTaggedPeerId, kbucketTag)
		}
	}
}

// Creates a new patcher instance
func NewPatcher() DHTPeerProtectionPatcher {
	return DHTPeerProtectionPatcher{
		MaxProtected:   0,
		ProtectionRate: .5,
		dist2protected: make(map[int]*orderedmap.OrderedMap),
		dist2tagged:    make(map[int]*orderedmap.OrderedMap),
	}
}

// Notify the patcher with a validated / known trusted peer id
// so that it will be prefered in the protected peer selection algorithm
func (p *DHTPeerProtectionPatcher) Heartbeat(peerId peer.ID) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	updated := false
	for _, protected := range p.dist2protected {
		if protected.Delete(peerId) {
			protected.Set(peerId, time.Now())
			updated = true
			break
		}
	}
	if !updated {
		for _, tagged := range p.dist2tagged {
			if tagged.Delete(peerId) {
				tagged.Set(peerId, time.Now())
				updated = true
				break
			}
		}
	}
	return updated
}

// Patches the peer protection algorithm of the given dht instance
func (p *DHTPeerProtectionPatcher) Patch(dht *kaddht.IpfsDHT) {
	p.dht = dht
	p.host = dht.Host()
	p.connMgr = p.host.ConnManager()
	p.selfId = kb.ConvertPeerID(dht.PeerID())
	p.routingTable = dht.RoutingTable()

	p.routingTable.PeerAdded = func(pid peer.ID) {
		p.connMgr.TagPeer(pid, kbucketTag, baseConnMgrScore)
		// Common prefix len is an approximate inverse of the kad-dht distance
		// Patcher aims to protect peers with minimal distance to us, hence we're protecting
		// peers with maximum kad-dht distance
		commonPrefixLen := kb.CommonPrefixLen(p.selfId, kb.ConvertPeerID(pid))
		p.lock.Lock()
		defer p.lock.Unlock()
		// TODO: Logic here can be more efficient
		// In reality, it's not likely to hold connections from a massive number of peers, say 100k+,
		// a naive implementation can be more readable and less error-prone
		insertThreadUnsafe(p.dist2tagged, commonPrefixLen, pid, time.UnixMicro(0))
		p.adjustProtectedThreadUnsafe()
	}

	peerRemoved := p.routingTable.PeerRemoved
	p.routingTable.PeerRemoved = func(pid peer.ID) {
		peerRemoved(pid)
		p.lock.Lock()
		defer p.lock.Unlock()
		deleted := false
		for _, protected := range p.dist2protected {
			if protected.Delete(pid) {
				deleted = true
				break
			}
		}
		if !deleted {
			for _, tagged := range p.dist2tagged {
				if tagged.Delete(pid) {
					break
				}
			}
		}
		// No need to call this since it does not decrease the protection rate yet
		// p.adjustProtectedThreadUnsafe()
	}
}

func insertThreadUnsafe(m map[int]*orderedmap.OrderedMap, distance int, id peer.ID, t time.Time) {
	om, ok := m[distance]
	if !ok {
		om = orderedmap.NewOrderedMap()
		m[distance] = om
	}
	om.Set(id, t)
}
