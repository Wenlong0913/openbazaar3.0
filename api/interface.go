package api

import (
	"context"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	peer "github.com/libp2p/go-libp2p-peer"
)

// CoreIface is used to get around a circular import of the Core package.
type CoreIface interface {
	RequestAddress(to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error)
	SendChatMessage(to peer.ID, message, subject string, done chan<- struct{}) error
	SendTypingMessage(to peer.ID, subject string) error
	MarkChatMessagesAsRead(peer peer.ID, subject string) error
	GetChatConversations() ([]models.ChatConversation, error)
	GetChatMessagesByPeer(peer peer.ID) ([]models.ChatMessage, error)
	GetChatMessagesBySubject(subject string) ([]models.ChatMessage, error)
	ConfirmOrder(orderID models.OrderID, done chan struct{}) error
	FollowNode(peerID peer.ID, done chan<- struct{}) error
	UnfollowNode(peerID peer.ID, done chan<- struct{}) error
	GetMyFollowers() (models.Followers, error)
	GetMyFollowing() (models.Following, error)
	GetFollowers(peerID peer.ID, useCache bool) (models.Followers, error)
	GetFollowing(peerID peer.ID, useCache bool) (models.Following, error)
	SaveListing(listing *pb.Listing, done chan<- struct{}) error
	DeleteListing(slug string, done chan<- struct{}) error
	GetMyListings() (models.ListingIndex, error)
	GetListings(peerID peer.ID, useCache bool) (models.ListingIndex, error)
	GetMyListingBySlug(slug string) (*pb.SignedListing, error)
	GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error)
	GetListingBySlug(peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error)
	GetListingByCID(cid cid.Cid) (*pb.SignedListing, error)
	SetSelfAsModerator(modInfo *models.ModeratorInfo, done chan struct{}) error
	RemoveSelfAsModerator(ctx context.Context, done chan<- struct{}) error
	GetModerators(ctx context.Context) []peer.ID
	GetModeratorsAsync(ctx context.Context) <-chan peer.ID
	Publish(done chan<- struct{})
	UsingTestnet() bool
	IPFSNode() *core.IpfsNode
	Identity() peer.ID
	SubscribeEvent(event interface{}) (events.Subscription, error)
	SetProfile(profile *models.Profile, done chan<- struct{}) error
	GetMyProfile() (*models.Profile, error)
	GetProfile(peerID peer.ID, useCache bool) (*models.Profile, error)
	PurchaseListing(purchase *models.Purchase) (orderID models.OrderID, paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error)
	EstimateOrderSubtotal(purchase *models.Purchase) (*models.CurrencyValue, error)
	RejectOrder(orderID models.OrderID, reason string, done chan struct{}) error
}