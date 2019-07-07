package bitcoincash

import (
	"encoding/json"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"time"
	"github.com/OpenBazaar/wallet-interface"
)

type httpClient interface {
	Get(string) (*http.Response, error)
}

type feeCache struct {
	fees        *Fees
	lastUpdated time.Time
}

type Fees struct {
	FastestFee  uint64
	HalfHourFee uint64
	HourFee     uint64
}

type FeeProvider struct {
	maxFee      uint64
	priorityFee uint64
	normalFee   uint64
	economicFee uint64
	feeAPI      string

	httpClient httpClient

	cache *feeCache
}

func NewFeeProvider(maxFee, priorityFee, normalFee, economicFee uint64, feeAPI string, proxy proxy.Dialer) *FeeProvider {
	fp := FeeProvider{
		maxFee:      maxFee,
		priorityFee: priorityFee,
		normalFee:   normalFee,
		economicFee: economicFee,
		feeAPI:      feeAPI,
		cache:       new(feeCache),
	}
	dial := net.Dial
	if proxy != nil {
		dial = proxy.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	httpClient := &http.Client{Transport: tbTransport, Timeout: time.Second * 10}
	fp.httpClient = httpClient
	return &fp
}

func (fp *FeeProvider) GetFeePerByte(feeLevel wallet.FeeLevel) uint64 {
	defaultFee := func() uint64 {
		switch feeLevel {
		case wallet.PRIOIRTY:
			return fp.priorityFee
		case wallet.NORMAL:
			return fp.normalFee
		case wallet.ECONOMIC:
			return fp.economicFee
		case wallet.FEE_BUMP:
			return fp.priorityFee * 2
		default:
			return fp.normalFee
		}
	}
	if fp.feeAPI == "" {
		return defaultFee()
	}
	fees := new(Fees)
	if time.Since(fp.cache.lastUpdated) > time.Minute {
		resp, err := fp.httpClient.Get(fp.feeAPI)
		if err != nil {
			return defaultFee()
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&fees)
		if err != nil {
			return defaultFee()
		}
		fp.cache.lastUpdated = time.Now()
		fp.cache.fees = fees
	} else {
		fees = fp.cache.fees
	}
	switch feeLevel {
	case wallet.PRIOIRTY:
		if fees.FastestFee > fp.maxFee || fees.FastestFee == 0 {
			return fp.maxFee
		} else {
			return fees.FastestFee
		}
	case wallet.NORMAL:
		if fees.HalfHourFee > fp.maxFee || fees.HalfHourFee == 0 {
			return fp.maxFee
		} else {
			return fees.HalfHourFee
		}
	case wallet.ECONOMIC:
		if fees.HourFee > fp.maxFee || fees.HourFee == 0 {
			return fp.maxFee
		} else {
			return fees.HourFee
		}
	case wallet.FEE_BUMP:
		if (fees.FastestFee*2) > fp.maxFee || fees.FastestFee == 0 {
			return fp.maxFee
		} else {
			return fees.FastestFee * 2
		}
	default:
		return fp.normalFee
	}
}
