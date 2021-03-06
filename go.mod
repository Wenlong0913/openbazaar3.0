module github.com/cpacia/openbazaar3.0

go 1.13

require (
	github.com/OpenBazaar/jsonpb v0.0.0-20171123000858-37d32ddf4eef
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/cpacia/go-libtor v1.0.138-0.20191212055005-1e3e913c0ef9
	github.com/cpacia/go-onion-transport v0.0.0-20191212094059-dd08e956eaa2
	github.com/cpacia/go-store-and-forward v0.0.0-20200121164237-a4a67c7eef82
	github.com/cpacia/multiwallet v0.0.0-20200229170431-cb00a6067c40
	github.com/cpacia/proxyclient v0.0.0-20191212063311-f03265f1fade
	github.com/cpacia/wallet-interface v0.0.0-20200229171801-8511c1248c5f
	github.com/cretz/bine v0.1.0
	github.com/disintegration/imaging v1.6.2
	github.com/fatih/color v1.7.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/gosimple/slug v1.6.0
	github.com/ipfs/go-bitswap v0.0.8-0.20200117195305-e37498cf10d6
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ipfs v0.4.23
	github.com/ipfs/go-ipfs-config v0.0.3
	github.com/ipfs/go-ipfs-files v0.0.3
	github.com/ipfs/go-ipns v0.0.1
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.0.3
	github.com/ipfs/go-path v0.0.4
	github.com/ipfs/interface-go-ipfs-core v0.0.8
	github.com/jarcoal/httpmock v1.0.4
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99
	github.com/jessevdk/go-flags v1.4.0
	github.com/jinzhu/gorm v1.9.11
	github.com/lib/pq v1.2.0 // indirect
	github.com/libp2p/go-libp2p v0.0.32
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/libp2p/go-libp2p-host v0.0.3
	github.com/libp2p/go-libp2p-kad-dht v0.0.15
	github.com/libp2p/go-libp2p-net v0.0.2
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/libp2p/go-libp2p-peerstore v0.0.6
	github.com/libp2p/go-libp2p-protocol v0.0.1
	github.com/libp2p/go-libp2p-record v0.0.1
	github.com/libp2p/go-libp2p-routing v0.0.1
	github.com/libp2p/go-msgio v0.0.4 // indirect
	github.com/libp2p/go-testutil v0.0.1
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/multiformats/go-multiaddr-net v0.0.1
	github.com/multiformats/go-multihash v0.0.10
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rainycape/unidecode v0.0.0-20150907023854-cb7f23ec59be // indirect
	github.com/tyler-smith/go-bip39 v1.0.2
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
)

replace (
	github.com/Roasbeef/ltcutil => github.com/ltcsuite/ltcutil v0.0.0-20181217130922-17f3b04680b6
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4-0.20200121170514-da442c51f155
	github.com/go-critic/go-critic => github.com/go-critic/go-critic v0.4.0
	github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.21.0
	github.com/lightninglabs/neutrino => github.com/lightninglabs/neutrino v0.11.0
)
