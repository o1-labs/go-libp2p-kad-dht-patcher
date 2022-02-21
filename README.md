# go-libp2p-kad-dht-patcher

Utility to patch peer protection logic in go-libp2p-kad-dht

## Options

```go
// Max number of peers to protect
// non-positive means unlimited
MaxProtected int
// Target percentage of protected peers, (0.0,1.0]
ProtectionRate float32
```

## Run tests

```bash
go mod vendor
go test -v
```
