# go-libp2p-kad-dht-patcher

Utility to patch peer protection logic in go-libp2p-kad-dht

## N-Bucket problem

Details to be added

## Run tests

```bash
go mod vendor
go test -v
```

## Options

```go
// Max number of peers to protect
// non-positive means unlimited
// default is 0
MaxProtected int
// Target percentage of protected peers, (0.0,1.0]
// default is 0.5
ProtectionRate float32
```

## Public APIs

```go
// Creates a new patcher instance
func NewPatcher() DHTPeerProtectionPatcher

// Notify the patcher with a validated / known trusted peer id
// so that it will be prefered in the protected peer selection algorithm
func (p *DHTPeerProtectionPatcher) Heartbeat(peerId peer.ID) bool

// Patches the peer protection algorithm of the given dht instance
func (p *DHTPeerProtectionPatcher) Patch(dht *kaddht.IpfsDHT)
```

## Usage

```go
patcher := NewPatcher()

if hostDHT, err := kaddht.New(ctx, host, dhtOpts...); err != nil {
    patcher.ProtectionRate = targetProtectionRate
	patcher.MaxProtected = maxProtected
	patcher.Patch(hostDHT)
}
```

Refer to `kbucket_fix_test.go` for details
