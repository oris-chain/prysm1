package state_native_test

import (
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestFieldTrie_NewTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 40)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	// 5 represents the enum value of state roots
	trie, err := state_native.NewFieldTrie(st, types.FieldIndex(5), types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}, newState.StateRoots(), uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
	require.NoError(t, err)
	root, err := stateutil.RootsArrayHashTreeRoot(newState.StateRoots(), uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
	require.NoError(t, err)
	newRoot, err := trie.TrieRoot()
	require.NoError(t, err)
	assert.Equal(t, root, newRoot)
}

func TestFieldTrie_NewTrie_NilElements(t *testing.T) {
	trie, err := state_native.NewFieldTrie(nil, types.FieldIndex(5), types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}, nil, 8234)
	require.NoError(t, err)
	_, err = trie.TrieRoot()
	require.ErrorIs(t, err, state_native.ErrEmptyFieldTrie)
}

func TestFieldTrie_RecomputeTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	// 10 represents the enum value of validators
	trie, err := state_native.NewFieldTrie(st, types.FieldIndex(11), types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}, newState.Validators(), params.BeaconConfig().ValidatorRegistryLimit)
	require.NoError(t, err)

	oldroot, err := trie.TrieRoot()
	require.NoError(t, err)
	require.NotEmpty(t, oldroot)

	changedIdx := []uint64{2, 29}
	val1, err := newState.ValidatorAtIndex(10)
	require.NoError(t, err)
	val2, err := newState.ValidatorAtIndex(11)
	require.NoError(t, err)
	val1.Slashed = true
	val1.ExitEpoch = 20

	val2.Slashed = true
	val2.ExitEpoch = 40

	changedVals := []*ethpb.Validator{val1, val2}
	require.NoError(t, newState.UpdateValidatorAtIndex(primitives.ValidatorIndex(changedIdx[0]), changedVals[0]))
	require.NoError(t, newState.UpdateValidatorAtIndex(primitives.ValidatorIndex(changedIdx[1]), changedVals[1]))

	expectedRoot, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(st, changedIdx, newState.Validators())
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
}

func TestFieldTrie_RecomputeTrie_CompressedArray(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	trie, err := state_native.NewFieldTrie(st, types.FieldIndex(12), types.FieldInfo{ArrayType: types.CompressedArray, ValueType: types.MultiValue}, state_native.NewMultiValueBalances(newState.Balances()), stateutil.ValidatorLimitForBalancesChunks())
	require.NoError(t, err)
	require.Equal(t, trie.Length(), stateutil.ValidatorLimitForBalancesChunks())
	changedIdx := []uint64{4, 8}
	require.NoError(t, newState.UpdateBalancesAtIndex(primitives.ValidatorIndex(changedIdx[0]), uint64(100000000)))
	require.NoError(t, newState.UpdateBalancesAtIndex(primitives.ValidatorIndex(changedIdx[1]), uint64(200000000)))
	expectedRoot, err := stateutil.Uint64ListRootWithRegistryLimit(newState.Balances())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(st, changedIdx, state_native.NewMultiValueBalances(newState.Balances()))
	require.NoError(t, err)

	// not equal for some reason :(
	assert.Equal(t, expectedRoot, root)
}

func TestNewFieldTrie_UnknownType(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	_, err := state_native.NewFieldTrie(st, types.FieldIndex(12), types.FieldInfo{ArrayType: 4, ValueType: types.MultiValue}, state_native.NewMultiValueBalances(newState.Balances()), 32)
	require.ErrorContains(t, "unrecognized data type", err)
}

func TestFieldTrie_CopyTrieImmutable(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	// 12 represents the enum value of randao mixes.
	trie, err := state_native.NewFieldTrie(st, types.FieldIndex(13), types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}, newState.RandaoMixes(), uint64(params.BeaconConfig().EpochsPerHistoricalVector))
	require.NoError(t, err)

	newTrie := trie.CopyTrie()

	changedIdx := []uint64{2, 29}

	changedVals := [][32]byte{{'A', 'B'}, {'C', 'D'}}
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[0], changedVals[0]))
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[1], changedVals[1]))

	root, err := trie.RecomputeTrie(st, changedIdx, newState.RandaoMixes())
	require.NoError(t, err)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	if root == newRoot {
		t.Errorf("Wanted roots to be different, but they are the same: %#x", root)
	}
}

func TestFieldTrie_CopyAndTransferEmpty(t *testing.T) {
	trie, err := state_native.NewFieldTrie(nil, types.FieldIndex(13), types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}, nil, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
	require.NoError(t, err)

	require.DeepEqual(t, trie, trie.CopyTrie())
	require.DeepEqual(t, trie, trie.TransferTrie())
}

func TestFieldTrie_TransferTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(t, true, ok)
	maxLength := (params.BeaconConfig().ValidatorRegistryLimit*8 + 31) / 32
	trie, err := state_native.NewFieldTrie(st, types.FieldIndex(12), types.FieldInfo{ArrayType: types.CompressedArray, ValueType: types.MultiValue}, state_native.NewMultiValueBalances(newState.Balances()), maxLength)
	require.NoError(t, err)
	oldRoot, err := trie.TrieRoot()
	require.NoError(t, err)

	newTrie := trie.TransferTrie()
	root, err := trie.TrieRoot()
	require.ErrorIs(t, err, state_native.ErrEmptyFieldTrie)
	require.Equal(t, root, [32]byte{})
	require.NotNil(t, newTrie)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	require.DeepEqual(t, oldRoot, newRoot)
}

func FuzzFieldTrie(f *testing.F) {
	newState, _ := util.DeterministicGenesisState(f, 40)
	st, ok := newState.(*state_native.BeaconState)
	require.Equal(f, true, ok)
	var data []byte
	for _, root := range newState.StateRoots() {
		data = append(data, root...)
	}
	f.Add(5, int(types.BasicArray), data, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))

	f.Fuzz(func(t *testing.T, idx, typ int, data []byte, slotsPerHistRoot uint64) {
		var roots [][]byte
		for i := 32; i < len(data); i += 32 {
			roots = append(roots, data[i-32:i])
		}
		trie, err := state_native.NewFieldTrie(st, types.FieldIndex(idx), types.FieldInfo{ArrayType: types.ArrayType(typ), ValueType: types.MultiValue}, roots, slotsPerHistRoot)
		if err != nil {
			return // invalid inputs
		}
		_, err = trie.TrieRoot()
		if err != nil {
			return
		}
	})
}