package chain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-filecoin/chain"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	tf "github.com/filecoin-project/go-filecoin/testhelpers/testflags"
	"github.com/filecoin-project/go-filecoin/types"
)

// getForkOldNewCommon
func getForkOldNewCommon(ctx context.Context, t *testing.T, chainStore chain.Store, blockSource *th.TestFetcher, dstP *DefaultSyncerTestParams) (types.TipSet, types.TipSet, types.TipSet) {
	// Add 10 tipsets to the head of the chainStore.
	requireGrowChain(ctx, t, blockSource, chainStore, 10, dstP)
	commonAncestor := requireHeadTipset(t, chainStore)

	// make the first fork tipset
	signer, ki := types.NewMockSignersAndKeyInfo(1)
	mockSignerPubKey := ki[0].PublicKey()
	fakeChildParams := th.FakeChildParams{
		Parent:      commonAncestor,
		GenesisCid:  dstP.genCid,
		Signer:      signer,
		MinerPubKey: mockSignerPubKey,
		StateRoot:   dstP.genStateRoot,
		Nonce:       uint64(4),
	}

	firstForkBlock := th.RequireMkFakeChild(t, fakeChildParams)
	requirePutBlocks(t, blockSource, firstForkBlock)
	firstForkTS := th.RequireNewTipSet(t, firstForkBlock)
	firstForkTsas := &chain.TipSetAndState{
		TipSet:          firstForkTS,
		TipSetStateRoot: dstP.genStateRoot,
	}
	th.RequirePutTsas(ctx, t, chainStore, firstForkTsas)
	err := chainStore.SetHead(ctx, firstForkTS)
	require.NoError(t, err)

	// grow the fork by 10 blocks (11 total)
	requireGrowChain(ctx, t, blockSource, chainStore, 10, dstP)
	oldHead := requireHeadTipset(t, chainStore)

	// go back and complete the original chain
	err = chainStore.SetHead(ctx, commonAncestor)
	require.NoError(t, err)
	requireGrowChain(ctx, t, blockSource, chainStore, 14, dstP)
	newHead := requireHeadTipset(t, chainStore)

	return oldHead, newHead, commonAncestor
}

func getPrefixOldNewCommon(ctx context.Context, t *testing.T, chainStore chain.Store, blockSource *th.TestFetcher, dstP *DefaultSyncerTestParams) (types.TipSet, types.TipSet, types.TipSet) {
	// Add 10 tipsets to the head of the chainStore.
	requireGrowChain(ctx, t, blockSource, chainStore, 10, dstP)
	oldHead := requireHeadTipset(t, chainStore)

	// Add 2 tipsets to the head
	requireGrowChain(ctx, t, blockSource, chainStore, 2, dstP)
	newHead := requireHeadTipset(t, chainStore)

	return oldHead, newHead, oldHead
}

func getSubsetOldNewCommon(ctx context.Context, t *testing.T, chainStore chain.Store, blockSource *th.TestFetcher, dstP *DefaultSyncerTestParams) (types.TipSet, types.TipSet, types.TipSet) {
	requireGrowChain(ctx, t, blockSource, chainStore, 10, dstP)
	commonAncestor := requireHeadTipset(t, chainStore)
	requireGrowChain(ctx, t, blockSource, chainStore, 1, dstP)
	oldHead := requireHeadTipset(t, chainStore)
	headBlock := oldHead.ToSlice()[0]

	signer, ki := types.NewMockSignersAndKeyInfo(1)
	mockSignerPubKey := ki[0].PublicKey()
	block2 := th.RequireMkFakeChild(t, th.FakeChildParams{
		Parent:      commonAncestor,
		MinerPubKey: mockSignerPubKey,
		Signer:      signer,
		GenesisCid:  dstP.genCid,
		StateRoot:   dstP.genStateRoot})
	requirePutBlocks(t, blockSource, block2)
	superset := th.RequireNewTipSet(t, headBlock, block2)
	supersetTsas := &chain.TipSetAndState{
		TipSet:          superset,
		TipSetStateRoot: dstP.genStateRoot,
	}
	th.RequirePutTsas(ctx, t, chainStore, supersetTsas)

	return oldHead, superset, commonAncestor
}

func TestIsReorgFork(t *testing.T) {
	tf.UnitTest(t)
	dstP := initDSTParams()
	ctx, blockSource, chainStore := setupGetAncestorTests(t, dstP)
	old, new, common := getForkOldNewCommon(ctx, t, chainStore, blockSource, dstP)
	assert.True(t, chain.IsReorg(old, new, common))
}
func TestIsReorgPrefix(t *testing.T) {
	tf.UnitTest(t)
	dstP := initDSTParams()
	ctx, blockSource, chainStore := setupGetAncestorTests(t, dstP)
	old, new, common := getPrefixOldNewCommon(ctx, t, chainStore, blockSource, dstP)
	assert.False(t, chain.IsReorg(old, new, common))
}

func TestIsReorgSubset(t *testing.T) {
	tf.UnitTest(t)
	dstP := initDSTParams()
	ctx, blockSource, chainStore := setupGetAncestorTests(t, dstP)
	old, new, common := getSubsetOldNewCommon(ctx, t, chainStore, blockSource, dstP)
	assert.False(t, chain.IsReorg(old, new, common))
}

func TestReorgDiffFork(t *testing.T) {
	tf.UnitTest(t)
	dstP := initDSTParams()
	ctx, blockSource, chainStore := setupGetAncestorTests(t, dstP)
	old, new, common := getForkOldNewCommon(ctx, t, chainStore, blockSource, dstP)

	dropped, added, err := chain.ReorgDiff(old, new, common)
	assert.NoError(t, err)
	assert.Equal(t, uint64(11), dropped)
	assert.Equal(t, uint64(14), added)
}

func TestReorgDiffSubset(t *testing.T) {
	tf.UnitTest(t)
	dstP := initDSTParams()
	ctx, blockSource, chainStore := setupGetAncestorTests(t, dstP)
	old, new, common := getSubsetOldNewCommon(ctx, t, chainStore, blockSource, dstP)

	dropped, added, err := chain.ReorgDiff(old, new, common)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), dropped)
	assert.Equal(t, uint64(1), added)
}
