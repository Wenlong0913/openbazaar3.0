package models

import (
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/ipfs/go-cid"
	"strings"
	"testing"
)

func TestListingIndex_UpdateListing(t *testing.T) {
	li := ListingIndex{}

	slug := "asdf"
	li.UpdateListing(ListingMetadata{
		Slug:  slug,
		Title: "abc",
	})

	exists := false
	for _, lm := range li {
		if lm.Slug == slug {
			exists = true
			break
		}
	}
	if !exists {
		t.Error("Failed to add listing metadata to index")
	}

	newTitle := "123"
	li.UpdateListing(ListingMetadata{
		Slug:  slug,
		Title: newTitle,
	})

	exists = false
	for _, lm := range li {
		if lm.Slug == slug {
			if lm.Title != newTitle {
				t.Error("Title failed to update")
			}
			exists = true
			break
		}
	}
	if !exists {
		t.Error("Failed to add listing metadata to index")
	}

}

func TestListingIndex_GetListingSlug(t *testing.T) {
	li := ListingIndex{}

	slug := "asdf"
	li.UpdateListing(ListingMetadata{
		Slug: slug,
		Hash: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
	})

	c, err := cid.Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}
	ret, err := li.GetListingSlug(c)
	if err != nil {
		t.Fatal(err)
	}

	if ret != slug {
		t.Errorf("Returned incorrect slug. Expected %s, got %s", slug, ret)
	}
}

func TestListingIndex_Count(t *testing.T) {
	li := ListingIndex{}

	slug := "asdf"
	li.UpdateListing(ListingMetadata{
		Slug: slug,
		Hash: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
	})

	ret := li.Count()
	if ret != 1 {
		t.Errorf("Returned incorrect count. Expected %d, got %d", 1, ret)
	}

}

func TestListingIndex_DeleteListing(t *testing.T) {
	li := ListingIndex{}

	slug := "asdf"
	li.UpdateListing(ListingMetadata{
		Slug: slug,
		Hash: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
	})

	li.DeleteListing(slug)

	ret := li.Count()
	if ret != 0 {
		t.Errorf("Returned incorrect count. Expected %d, got %d", 0, ret)
	}
}

func TestNewListingMetadataFromListing(t *testing.T) {
	c, err := cid.Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}

	listing := &pb.Listing{
		Slug: "abc",
		Item: &pb.Listing_Item{
			Title: strings.Repeat("s", ShortDescriptionLength+1),
			Price: "1000",
			Images: []*pb.Listing_Item_Image{
				{
					Tiny:   "aaa",
					Small:  "bbb",
					Medium: "ccc",
				},
			},
		},
		Metadata: &pb.Listing_Metadata{
			PricingCurrency: &pb.Currency{
				Code: "BTC",
			},
		},
		ShippingOptions: []*pb.Listing_ShippingOption{
			{
				Regions: []pb.CountryCode{
					pb.CountryCode_ALBANIA,
				},
				Services: []*pb.Listing_ShippingOption_Service{
					{
						Price: "0",
						Name:  "asdf",
					},
				},
			},
		},
	}

	ret, err := NewListingMetadataFromListing(listing, c)
	if err != nil {
		t.Fatal(err)
	}

	if len(ret.ShipsTo) != 1 {
		t.Errorf("Returned incorrect shipping regions. Expected %d, got %d", 1, len(ret.ShipsTo))
	}

	if len(ret.FreeShipping) != 1 {
		t.Errorf("Returned incorrect shipping regions. Expected %d, got %d", 1, len(ret.FreeShipping))
	}
}
