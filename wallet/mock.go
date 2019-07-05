package wallet

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btc "github.com/btcsuite/btcutil"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type MockWalletNetwork struct {
	wallets []MockWallet
}

func (n *MockWalletNetwork) GenerateBlock() {
	h := make([]byte, 32)
	rand.Read(h)

	ch, _ := chainhash.NewHash(h)
	for _, wallet := range n.wallets {
		wallet.block <- *ch
	}
}

func (n *MockWalletNetwork) GenerateToAddress(addr btc.Address, amount big.Int) {

}

type MockWallet struct {
	mtx sync.RWMutex

	addrs        map[string]bool
	watchedAddrs map[string]struct{}
	transactions map[chainhash.Hash]wallet.Txn
	utxos        map[string]mockUtxo

	listeners []func(wallet.TransactionCallback)

	bestHeight int
	bestHash   chainhash.Hash

	outgoing chan wallet.TransactionCallback
	incoming chan wallet.TransactionCallback
	block    chan chainhash.Hash

	done chan struct{}
}

type mockUtxo struct {
	OutpointHash  []byte
	OutpointIndex uint32
	Address       btc.Address
	Value         big.Int
	Height        int
}

// Start is called when the openbazaar-go daemon starts up. At this point in time
// the wallet implementation should start syncing and/or updating balances, but
// not before.
func (w *MockWallet) Start() {
	go func() {
		for {
			select {
			case tx := <-w.incoming:
				w.mtx.Lock()
				shouldCallback := false
				var value int64
				txid, _ := chainhash.NewHashFromStr(tx.Txid)
				txidBytes, _ := hex.DecodeString(tx.Txid)
				for _, in := range tx.Inputs {
					if _, ok := w.utxos[hex.EncodeToString(in.OutpointHash)+strconv.Itoa(int(in.OutpointIndex))]; ok {
						shouldCallback = true
						value -= in.Value
						delete(w.utxos, hex.EncodeToString(in.OutpointHash)+strconv.Itoa(int(in.OutpointIndex)))
					}
				}
				for i, out := range tx.Outputs {
					if _, ok := w.addrs[out.Address.String()]; ok {
						shouldCallback = true
						value += out.Value
						if _, ok := w.transactions[*txid]; !ok {
							w.utxos[tx.Txid+strconv.Itoa(i)] = mockUtxo{
								Value:         *big.NewInt(out.Value),
								Address:       out.Address,
								OutpointHash:  txidBytes,
								OutpointIndex: uint32(i),
							}
						}
					}
				}
				if shouldCallback {
					for _, listener := range w.listeners {
						listener(tx)
					}
					txn := wallet.Txn{
						Timestamp: time.Now(),
						Txid:      txid.String(),
						Value:     value,
						WatchOnly: value == 0,
					}
					w.transactions[*txid] = txn
				}
				w.mtx.Unlock()
			case blockHash := <-w.block:
				w.mtx.Lock()
				w.bestHeight++
				w.bestHash = blockHash

				for id, utxo := range w.utxos {
					if utxo.Height == 0 {
						utxo.Height = w.bestHeight
						w.utxos[id] = utxo
					}
				}
				for h, txn := range w.transactions {
					if txn.Height == 0 {
						txn.Height = int32(w.bestHeight)
						w.transactions[h] = txn
					}
				}
				w.mtx.Unlock()
			case <-w.done:
				return
			}
		}
	}()
}

// Close should cleanly disconnect from the wallet and finish writing
// anything it needs to to disk.
func (w *MockWallet) Close() {
	close(w.done)
}

// CurrencyCode returns the currency code this wallet implements. For example, "BTC".
// When running on testnet a `T` should be prepended. For example "TBTC".
func (w *MockWallet) CurrencyCode() string {
	return "TMCK"
}

// ExchangeRates returns an ExchangeRates implementation which will provide
// fiat exchange rate data for this coin.
func (w *MockWallet) ExchangeRates() wallet.ExchangeRates {

}

// AddWatchedAddress adds an address to the wallet to get notifications back when coins
// are received or spent from it. These watch only addresses should be persisted between
// sessions and upon each startup the wallet should be made to listen for transactions
// involving them.
func (w *MockWallet) AddWatchedAddress(addr btc.Address) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.watchedAddrs[addr.String()] = struct{}{}
	return nil
}

// AddTransactionListener is how openbazaar-go registers to receive a callback whenever
// a transaction is received that is relevant to this wallet or any of its watch only
// addresses. An address is considered relevant if any inputs or outputs match an address
// owned by this wallet, or being watched by the wallet via AddWatchedAddress method.
func (w *MockWallet) AddTransactionListener(cb func(wallet.TransactionCallback)) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.listeners = append(w.listeners, cb)
}

// IsDust returns whether the amount passed in is considered dust by network. This
// method is called when building payout transactions from the multisig to the various
// participants. If the amount that is supposed to be sent to a given party is below
// the dust threshold, openbazaar-go will not pay that party to avoid building a transaction
// that never confirms.
func (w *MockWallet) IsDust(amount big.Int) bool {
	return amount.Cmp(big.NewInt(100)) < 0
}

// CurrentAddress returns an address suitable for receiving payments. `purpose` specifies
// whether the address should be internal or external. External addresses are typically
// requested when receiving funds from outside the wallet .Internal addresses are typically
// change addresses. For utxo based coins we expect this function will return the same
// address so long as that address is unused. Whenever the address receives a payment,
// CurrentAddress should start returning a new, unused address.
func (w *MockWallet) CurrentAddress(purpose wallet.KeyPurpose) btc.Address {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	for addrStr, used := range w.addrs {
		if !used {
			addr, _ := btc.DecodeAddress(addrStr, &chaincfg.TestNet3Params)
			return addr
		}
	}
	h := make([]byte, 20)
	rand.Read(h)
	addr, _ := btc.NewAddressPubKeyHash(h, &chaincfg.TestNet3Params)
	w.addrs[addr.String()] = false
	return addr
}

// NewAddress returns a new, never-before-returned address. It is critical that it returns
// a never-before-returned address because this function is called when fetching an address
// for a direct payment order. In this case we expect the address to be unique for each order
// if it's not unique, it will cause problems as we can't determine which order the payment
// was for.
func (w *MockWallet) NewAddress(purpose wallet.KeyPurpose) btc.Address {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	h := make([]byte, 20)
	rand.Read(h)
	addr, _ := btc.NewAddressPubKeyHash(h, &chaincfg.TestNet3Params)
	w.addrs[addr.String()] = false
	return addr
}

// DecodeAddress parses the address string and return an address interface.
func (w *MockWallet) DecodeAddress(addr string) (btc.Address, error) {
	return btc.DecodeAddress(addr, &chaincfg.TestNet3Params)
}

// ScriptToAddress takes a raw output script (the full script, not just a hash160) and
// returns the corresponding address. This should be considered deprecated as we
// intend to remove it once most people have upgraded, but for now it needs to remain.
func (w *MockWallet) ScriptToAddress(script []byte) (btc.Address, error) {
	return nil, nil
}

// Balance returns the confirmed and unconfirmed aggregate balance for the wallet.
// For utxo based wallets, if a spend of confirmed coins is made, the resulting "change"
// should be also counted as confirmed even if the spending transaction is unconfirmed.
// The reason for this that if the spend never confirms, no coins will be lost to the wallet.
//
// The returned balances should be in the coin's base unit (for example: satoshis)
func (w *MockWallet) Balance() (confirmed, unconfirmed big.Int) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	for _, utxo := range w.utxos {
		if utxo.Height == 0 {
			unconfirmed.Add(&unconfirmed, &utxo.Value)
		} else {
			confirmed.Add(&confirmed, &utxo.Value)
		}
	}
	return
}

// Transactions returns a list of transactions for this wallet.
func (w *MockWallet) Transactions() ([]wallet.Txn, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	var txns []wallet.Txn
	for _, txn := range w.transactions {
		txns = append(txns, txn)
	}
	return txns, nil
}

// GetTransaction return info on a specific transaction given the txid.
func (w *MockWallet) GetTransaction(txid chainhash.Hash) (wallet.Txn, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	txn, ok := w.transactions[txid]
	if !ok {
		return wallet.Txn{}, errors.New("not found")
	}
	return txn, nil
}

// ChainTip returns the best block hash and height of the blockchain.
func (w *MockWallet) ChainTip() (uint32, chainhash.Hash) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	return uint32(w.bestHeight), w.bestHash
}

// ReSyncBlockchain is called in response to a user action to rescan transactions. API based
// wallets should do another scan of their addresses to find anything missing. Full node, or SPV
// wallets should rescan/re-download blocks starting at the fromTime.
func (w *MockWallet) ReSyncBlockchain(fromTime time.Time) {}

// GetConfirmations returns the number of confirmations and the height for a transaction.
func (w *MockWallet) GetConfirmations(txid chainhash.Hash) (confirms, atHeight uint32, err error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	txn, ok := w.transactions[txid]
	if !ok {
		return 0, 0, errors.New("not found")
	}
	return uint32(w.bestHeight) - uint32(txn.Height) + 1, uint32(txn.Height), nil
}

// ChildKey generate a child key using the given chaincode. Each openbazaar-go node
// keeps a master key (an hd secp256k1 key) that it uses in multisig transactions.
// Rather than use the key directly (which would result in an on chain privacy leak),
// we create a random chaincode for each order (which is not made public) and a child
// key is derived from the master key using the chaincode. The child key for each party
// to the order (buyer, vendor, moderator) is what is used to create the multisig. This
// function leaves it up the wallet implementation to decide how to derive the child key
// so long as it's deterministic and uses the chaincode and the returned key is pseudorandom.
func (w *MockWallet) ChildKey(keyBytes []byte, chaincode []byte, isPrivateKey bool) (*hd.ExtendedKey, error) {
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	var id []byte
	if isPrivateKey {
		id = chaincfg.TestNet3Params.HDPrivateKeyID[:]
	} else {
		id = chaincfg.TestNet3Params.HDPublicKeyID[:]
	}
	hdKey := hd.NewExtendedKey(
		id,
		keyBytes,
		chaincode,
		parentFP,
		0,
		0,
		isPrivateKey)
	return hdKey.Child(0)
}

// HasKey returns whether or not the wallet has the key for the given address. This method
// is called by openbazaar-go when validating payouts from multisigs. It makes sure the
// transaction that the other party(s) signed does indeed pay to an address that we
// control.
func (w *MockWallet) HasKey(addr btc.Address) bool {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	_, ok := w.addrs[addr.String()]
	return ok
}

// GenerateMultisigScript should deterministically create a redeem script and address from the information provided.
// This method should be strictly limited to taking the input data, combining it to produce the redeem script and
// address and that's it. There is no need to interact with the network or make any transactions when this is called.
//
// Openbazaar-go will call this method in the following situations:
// 1) When the buyer places an order he passes in the relevant keys for each party to get back the address where
// the funds should be sent and the redeem script. The redeem script is saved in order (and openbazaar-go database).
//
// 2) The vendor calls this method when he receives and order so as to validate that the address they buyer is sending
// funds to is indeed correctly constructed. If this method fails to return the same values for the vendor as it
// did the buyer, the vendor will reject the order.
//
// 3) The moderator calls this function upon receiving a dispute so that he can validate the payment address for the
// order and make sure neither party is trying to maliciously lie about the details of the dispute to get the moderator
// to release the funds.
//
// Note that according to the order flow, this method is called by the buyer *before* the order is sent to the vendor,
// and before the vendor validates the order. Only after the buyer hears back from the vendor does the buyer send
// funds (either from an external wallet or via the `Spend` method) to the address specified in this method's return.
//
// `threshold` is the number of keys required to release the funds from the address. If `threshold` is two and len(keys)
// is three, this is a two of three multisig. If `timeoutKey` is not nil, then the script should allow the funds to
// be released with a signature from the `timeoutKey` after the `timeout` duration has passed.
// For example:
// OP_IF 2 <buyerPubkey> <vendorPubkey> <moderatorPubkey> 3 OP_ELSE <timeout> OP_CHECKSEQUENCEVERIFY <timeoutKey> OP_CHECKSIG OP_ENDIF
//
// If `timeoutKey` is nil then the a normal multisig without a timeout should be created.
func (w *MockWallet) GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (addr btc.Address, redeemScript []byte, err error) {
	var mockReedemScript []byte
	for _, key := range keys {
		pub, err := key.ECPubKey()
		if err != nil {
			return nil, nil, err
		}
		mockReedemScript = append(mockReedemScript, pub.SerializeCompressed()...)
	}

	addr, err = btc.NewAddressScriptHash(mockReedemScript, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, nil, err
	}
	return addr, redeemScript, err
}

// CreateMultisigSignature should build a transaction using the given inputs and outputs and sign it with the
// provided key. A list of signatures (one for each input) should be returned.
//
// This method is called by openbazaar-go by each party whenever they decide to release the funds from escrow.
// This method should not actually move any funds or make any transactions, only create necessary signatures to
// do so. The caller will then take the signature and share it with the other parties. Once all parties have shared
// their signatures, the person who wants to release the funds collects them and uses them as an input to the
// `Multisign` method.
func (w *MockWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte big.Int) ([]wallet.Signature, error) {
	var sigs []wallet.Signature
	for i, in := range ins {
		sigBytes := append(in.OutpointHash, in.OrderID...)
		sigs = append(sigs, wallet.Signature{
			Signature:  sigBytes,
			InputIndex: uint32(i),
		})
	}
	return sigs, nil
}

// Multisign collects all of the signatures generated by the `CreateMultisigSignature` function and builds a final
// transaction that can then be broadcast to the blockchain. The []byte return is the raw transaction. It should be
// broadcasted if `broadcast` is true. If the signatures combine and produce an invalid transaction then an error
// should be returned.
//
// This method is called by openbazaar-go by whichever party to the escrow is trying to release the funds only after
// all needed parties have signed using `CreateMultisigSignature` and have shared their signatures with each other.
func (w *MockWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte big.Int, broadcast bool) ([]byte, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var mockTx []byte
	var cbin []wallet.TransactionInput
	for _, in := range ins {
		mockTx = append(mockTx, in.OutpointHash...)
		cbin = append(cbin, wallet.TransactionInput{
			Value:         in.Value,
			OutpointHash:  in.OutpointHash,
			OutpointIndex: in.OutpointIndex,
			LinkedAddress: in.LinkedAddress,
		})
	}
	var value int64 // TODO: switch to big.Int once new interface is merged.
	var cbout []wallet.TransactionOutput
	for _, out := range outs {
		if _, ok := w.addrs[out.Address.String()]; ok {
			value++
		}
		mockTx = append(mockTx, out.Address.ScriptAddress()...)
		cbout = append(cbout, wallet.TransactionOutput{
			Value:   out.Value,
			Address: out.Address,
			OrderID: out.OrderID,
			Index:   out.Index,
		})
	}
	for _, sig := range sigs1 {
		mockTx = append(mockTx, sig.Signature...)
	}
	for _, sig := range sigs2 {
		mockTx = append(mockTx, sig.Signature...)
	}
	txid := chainhash.DoubleHashH(mockTx)

	for i, out := range cbout {
		txidBytes, err := hex.DecodeString(txid.String())
		if err != nil {
			return nil, err
		}
		if _, ok := w.addrs[out.Address.String()]; ok {
			w.utxos[txid.String()+strconv.Itoa(i)] = mockUtxo{
				OutpointIndex: uint32(i),
				OutpointHash:  txidBytes,
				Value:         *big.NewInt(out.Value),
				Address:       out.Address,
			}
		}
	}

	txn := wallet.Txn{
		Bytes:     mockTx,
		Timestamp: time.Now(),
		Txid:      txid.String(),
		Value:     value,
		WatchOnly: value == 0,
	}

	cb := wallet.TransactionCallback{
		Value:     value,
		Timestamp: time.Now(),
		Txid:      txid.String(),
		WatchOnly: value == 0,
		Inputs:    cbin,
		Outputs:   cbout,
	}

	for _, listener := range w.listeners {
		listener(cb)
	}

	w.transactions[txid] = txn
	if w.outgoing != nil {
		w.outgoing <- cb
	}

	return mockTx, nil
}

// GetFeePerByte returns the current fee per byte for the given fee level. There
// are three fee levels ― priority, normal, and economic.
//
//The returned value should be in the coin's base unit (for example: satoshis).
func (w *MockWallet) GetFeePerByte(feeLevel wallet.FeeLevel) big.Int {
	return *big.NewInt(1)
}

// Spend transfers the given amount of coins (in the coin's base unit. For example: in
// satoshis) to the given address using the provided fee level. Openbazaar-go calls
// this method in two places. 1) When the user requests a normal transfer from their
// wallet to another address. 2) When clicking 'pay from internal wallet' to fund
// an order the user just placed.
// It also includes a referenceID which basically refers to the order the spend will affect
//
// If spendAll is true the amount field will be ignored and all the funds in the wallet will
// be swept to the provided payment address. For most coins this entails subtracting the
// transaction fee from the total amount being sent rather than adding it on as is normally
// the case when spendAll is false.
func (w *MockWallet) Spend(amount big.Int, addr btc.Address, feeLevel wallet.FeeLevel, referenceID string, spendAll bool) (*chainhash.Hash, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var utxos []mockUtxo
	total := new(big.Int)
	total.Add(total, big.NewInt(250))
	var cbin []wallet.TransactionInput
	for _, utxo := range w.utxos {
		utxos = append(utxos, utxo)
		total.Add(total, &utxo.Value)
		cbin = append(cbin, wallet.TransactionInput{
			Value:         utxo.Value.Int64(),
			LinkedAddress: utxo.Address,
			OutpointHash:  utxo.OutpointHash,
			OutpointIndex: utxo.OutpointIndex,
		})

		if total.Cmp(&amount) >= 0 {
			break
		}
	}
	if total.Cmp(&amount) < 0 {
		return nil, wallet.ErrorInsuffientFunds
	}

	for _, utxo := range utxos {
		delete(w.utxos, hex.EncodeToString(utxo.OutpointHash)+strconv.Itoa(int(utxo.OutpointIndex)))
	}

	change := total.Sub(total, &amount)

	cbout := []wallet.TransactionOutput{
		{
			Address: addr,
			Value:   amount.Int64(),
			Index:   0,
			OrderID: referenceID,
		},
	}

	if change.Cmp(big.NewInt(0)) >= 0 {
		changeAddr := w.NewAddress(wallet.INTERNAL)
		id := make([]byte, 32)
		rand.Read(id)
		w.utxos[hex.EncodeToString(id)+strconv.Itoa(1)] = mockUtxo{
			OutpointHash:  id,
			OutpointIndex: 1,
			Value:         *change,
			Address:       changeAddr,
		}
		cbout = append(cbout, wallet.TransactionOutput{
			Address: changeAddr,
			Value:   change.Int64(),
			Index:   1,
			OrderID: referenceID,
		})
	}

	value := big.NewInt(0)
	value = value.Sub(value, &amount)

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)
	txid, err := chainhash.NewHash(txidBytes)
	if err != nil {
		return nil, err
	}

	txn := wallet.Txn{
		Value:     value.Int64(),
		Txid:      txid.String(),
		Timestamp: time.Now(),
	}

	cb := wallet.TransactionCallback{
		Value:     value.Int64(),
		Txid:      txid.String(),
		Timestamp: time.Now(),
		Inputs:    cbin,
		Outputs:   cbout,
	}

	w.transactions[*txid] = txn

	for _, listener := range w.listeners {
		listener(cb)
	}

	if w.outgoing != nil {
		w.outgoing <- cb
	}

	return txid, nil

}

// EstimateFee should return the estimate fee that will be required to make a transaction
// spending from the given inputs to the given outputs. FeePerByte is denominated in
// the coin's base unit (for example: satoshis).
func (w *MockWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte big.Int) big.Int {
	return *big.NewInt(250)
}

// EstimateSpendFee should return the anticipated fee to transfer a given amount of coins
// out of the wallet at the provided fee level. Typically this involves building a
// transaction with enough inputs to cover the request amount and calculating the size
// of the transaction. It is OK, if a transaction comes in after this function is called
// that changes the estimated fee as it's only intended to be an estimate.
//
// All amounts should be in the coin's base unit (for example: satoshis).
func (w *MockWallet) EstimateSpendFee(amount big.Int, feeLevel wallet.FeeLevel) (big.Int, error) {
	return *big.NewInt(250), nil
}

// SweepAddress should sweep all the funds from the provided inputs into the provided `address` using the given
// `key`. If `address` is nil, the funds should be swept into an internal address own by this wallet.
// If the `redeemScript` is not nil, this should be treated as a multisig (p2sh) address and signed accordingly.
//
// This method is called by openbazaar-go in the following scenarios:
// 1) The buyer placed a direct order to a vendor who was offline. The buyer sent funds into a 1 of 2 multisig.
// Upon returning online the vendor accepts the order and calls SweepAddress to move the funds into his wallet.
//
// 2) Same as above but the buyer wishes to cancel the order before the vendor comes online. He calls SweepAddress
// to return the funds from the 1 of 2 multisig back into has wallet.
//
// 3) Same as above but rather than accepting the order, the vendor rejects it. When the buyer receives the reject
// message he calls SweepAddress to move the funds back into his wallet.
//
// 4) The timeout has expired on a 2 of 3 multisig. The vendor calls SweepAddress to claim the funds.
func (w *MockWallet) SweepAddress(ins []wallet.TransactionInput, address *btc.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {

}

// BumpFee should attempt to bump the fee on a given unconfirmed transaction (if possible) to
// try to get it confirmed and return the txid of the new transaction (if one exists).
// Since this method is only called in response to user action, it is acceptable to
// return an error if this functionality is not available in this wallet or on the network.
func (w *MockWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	return nil, nil
}