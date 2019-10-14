package orders

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"reflect"
	"testing"
)

func TestOrderProcessor_processRefundMessage(t *testing.T) {
	op, err := newMockOrderProcessor()
	if err != nil {
		t.Fatal(err)
	}

	wn := wallet.NewMockWalletNetwork(1)
	go wn.Start()

	addr, err := wn.Wallets()[0].CurrentAddress()
	if err != nil {
		t.Fatal(err)
	}
	addr2, err := wn.Wallets()[0].NewAddress()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(addr, addr2)
	if err := wn.GenerateToAddress(addr, iwallet.NewAmount(100000)); err != nil {
		t.Fatal(err)
	}

	wdbtx, err := wn.Wallets()[0].Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = wn.Wallets()[0].Spend(wdbtx, addr2, iwallet.NewAmount(1000), iwallet.FlNormal)
	if err != nil {
		t.Fatal(err)
	}

	if err := wdbtx.Commit(); err != nil {
		t.Fatal(err)
	}

	txs, err := wn.Wallets()[0].Transactions()
	if err != nil {
		t.Fatal(err)
	}

	op.multiwallet["TMCK"] = wn.Wallets()[0]

	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	remotePeer, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	refundMsg := &pb.Refund{
		TransactionID: txs[1].ID.String(),
	}

	refundAny, err := ptypes.MarshalAny(refundMsg)
	if err != nil {
		t.Fatal(err)
	}

	orderMsg := &npb.OrderMessage{
		OrderID:     "1234",
		MessageType: npb.OrderMessage_REFUND,
		Message:     refundAny,
	}

	var (
		vendorPeerID   = "xyz"
		vendorHandle   = "abc"
		smallImageHash = "aaaa"
		tinyImageHash  = "bbbb"
	)
	orderOpen := &pb.OrderOpen{
		Listings: []*pb.SignedListing{
			{
				Listing: &pb.Listing{
					VendorID: &pb.ID{
						PeerID: vendorPeerID,
						Handle: vendorHandle,
					},
					Item: &pb.Listing_Item{
						Images: []*pb.Listing_Item_Image{
							{
								Small: smallImageHash,
								Tiny:  tinyImageHash,
							},
						},
					},
				},
			},
		},
		Payment: &pb.OrderOpen_Payment{
			Coin:    "TMCK",
			Address: addr.String(),
		},
	}

	tests := []struct {
		setup         func(order *models.Order) error
		expectedError error
		expectedEvent interface{}
		checkTxs      func(order *models.Order) error
	}{
		{
			// Normal case where order open exists.
			setup: func(order *models.Order) error {
				order.ID = "1234"
				order.PaymentAddress = addr.String()
				return order.PutMessage(orderOpen)
			},
			expectedError: nil,
			expectedEvent: &events.RefundNotification{
				OrderID: "1234",
				Thumbnail: events.Thumbnail{
					Tiny:  tinyImageHash,
					Small: smallImageHash,
				},
				VendorHandle: vendorHandle,
				VendorID:     vendorPeerID,
			},
			checkTxs: func(order *models.Order) error {
				orderTxs, err := order.GetTransactions()
				if err != nil {
					return err
				}
				if len(orderTxs) == 0 {
					return errors.New("failed to record any tx")
				}
				if orderTxs[0].ID != txs[1].ID {
					return errors.New("failed to record tx")
				}
				return nil
			},
		},
		{
			// Duplicate order refund.
			setup: func(order *models.Order) error {
				return order.PutMessage(refundMsg)
			},
			expectedError: nil,
			expectedEvent: nil,
			checkTxs: func(order *models.Order) error {
				return nil
			},
		},
		{
			// Duplicate but different.
			setup: func(order *models.Order) error {
				msg2 := *refundMsg
				msg2.TransactionID = "abc"
				return order.PutMessage(&msg2)
			},
			expectedError: ErrChangedMessage,
			expectedEvent: nil,
			checkTxs: func(order *models.Order) error {
				return nil
			},
		},
		{
			// Out of order.
			setup: func(order *models.Order) error {
				order.SerializedOrderOpen = nil
				return nil
			},
			expectedError: nil,
			expectedEvent: nil,
			checkTxs: func(order *models.Order) error {
				return nil
			},
		},
	}

	for i, test := range tests {
		order := &models.Order{}
		if err := test.setup(order); err != nil {
			t.Errorf("Test %d setup error: %s", i, err)
			continue
		}
		err := op.db.Update(func(tx database.Tx) error {
			event, err := op.processRefundMessage(tx, order, remotePeer, orderMsg)
			if err != test.expectedError {
				return fmt.Errorf("incorrect error returned. Expected %t, got %t", test.expectedError, err)
			}
			if !reflect.DeepEqual(event, test.expectedEvent) {
				return fmt.Errorf("incorrect event returned")
			}
			return test.checkTxs(order)
		})
		if err != nil {
			t.Errorf("Error executing db update in test %d: %s", i, err)
		}
	}
}