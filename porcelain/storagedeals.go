package porcelain

import (
	"context"

	"github.com/ipfs/go-cid"
	errors "github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/plumbing/strgdls"
	"github.com/filecoin-project/go-filecoin/protocol/storage/storagedeal"
)

var (
	// ErrDealNotFound means DealGet failed to find a matching deal
	ErrDealNotFound = errors.New("deal not found")
)

type dealGetPlumbing interface {
	DealsLs(context.Context) (<-chan *strgdls.StorageDealLsResult, error)
}

// DealGet returns a single deal matching a given cid or an error
func DealGet(ctx context.Context, plumbing dealGetPlumbing, dealCid cid.Cid) (*storagedeal.Deal, error) {
	dealCh, err := plumbing.DealsLs(ctx)
	if err != nil {
		return nil, err
	}
	for deal := range dealCh {
		if deal.Err != nil {
			return nil, deal.Err
		}
		if deal.Deal.Response.ProposalCid == dealCid {
			return &deal.Deal, nil
		}
	}
	return nil, ErrDealNotFound
}

type dealClientLsPlumbing interface {
	ConfigGet(string) (interface{}, error)
	DealsLs(context.Context) (<-chan *strgdls.StorageDealLsResult, error)
}

// DealClientLs returns a channel with all deals placed as a client
func DealClientLs(ctx context.Context, plumbing dealClientLsPlumbing) (<-chan *strgdls.StorageDealLsResult, error) {
	minerAddress, _ := plumbing.ConfigGet("mining.minerAddress")

	dealCh, err := plumbing.DealsLs(ctx)
	if err != nil {
		return nil, err
	}

	outCh := filterDealChannel(dealCh, func(deal *storagedeal.Deal) bool {
		return deal.Miner != minerAddress
	})

	return outCh, nil
}

type dealMinerLsPlumbing interface {
	ConfigGet(string) (interface{}, error)
	DealsLs(context.Context) (<-chan *strgdls.StorageDealLsResult, error)
}

// DealMinerLs returns a channel with all deals received as a miner
func DealMinerLs(ctx context.Context, plumbing dealMinerLsPlumbing) (<-chan *strgdls.StorageDealLsResult, error) {
	minerAddress, _ := plumbing.ConfigGet("mining.minerAddress")

	dealCh, err := plumbing.DealsLs(ctx)
	if err != nil {
		return nil, err
	}

	outCh := filterDealChannel(dealCh, func(deal *storagedeal.Deal) bool {
		return deal.Miner == minerAddress
	})

	return outCh, nil
}

func filterDealChannel(dealCh <-chan *strgdls.StorageDealLsResult, filterFunc func(*storagedeal.Deal) bool) <-chan *strgdls.StorageDealLsResult {
	outCh := make(chan *strgdls.StorageDealLsResult)

	go func() {
		defer close(outCh)
		for deal := range dealCh {
			if deal.Err != nil || filterFunc(&deal.Deal) {
				outCh <- deal
			}
		}
	}()

	return outCh
}
