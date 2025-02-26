/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protoutil_test

import (
	"crypto/sha256"
	"encoding/asn1"
	"math"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	cb "github.com/hyperledger/fabric-protos-go/common"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testChannelID = "myuniquetestchainid"

func TestNewBlock(t *testing.T) {
	var block *cb.Block
	assert.Nil(t, block.GetHeader())
	assert.Nil(t, block.GetData())
	assert.Nil(t, block.GetMetadata())

	data := &cb.BlockData{
		Data: [][]byte{{0, 1, 2}},
	}
	block = protoutil.NewBlock(uint64(0), []byte("datahash"))
	assert.Equal(t, []byte("datahash"), block.Header.PreviousHash, "Incorrect previous hash")
	assert.NotNil(t, block.GetData())
	assert.NotNil(t, block.GetMetadata())
	block.GetHeader().DataHash = protoutil.BlockDataHash(data,block.Header.Number)

	asn1Bytes, err := asn1.Marshal(struct {
		Number       int64
		PreviousHash []byte
		DataHash     []byte
	}{
		Number:       0,
		DataHash:     protoutil.BlockDataHash(data,block.Header.Number),
		PreviousHash: []byte("datahash"),
	})
	headerHash := sha256.Sum256(asn1Bytes)
	assert.NoError(t, err)
	assert.Equal(t, asn1Bytes, protoutil.BlockHeaderBytes(block.Header), "Incorrect marshaled blockheader bytes")
	assert.Equal(t, headerHash[:], protoutil.BlockHeaderHash(block.Header), "Incorrect blockheader hash")
}

func TestGoodBlockHeaderBytes(t *testing.T) {
	goodBlockHeader := &common.BlockHeader{
		Number:       1,
		PreviousHash: []byte("foo"),
		DataHash:     []byte("bar"),
	}

	_ = protoutil.BlockHeaderBytes(goodBlockHeader) // Should not panic

	goodBlockHeaderMaxNumber := &common.BlockHeader{
		Number:       math.MaxUint64,
		PreviousHash: []byte("foo"),
		DataHash:     []byte("bar"),
	}

	_ = protoutil.BlockHeaderBytes(goodBlockHeaderMaxNumber) // Should not panic
}

func TestGetChannelIDFromBlockBytes(t *testing.T) {
	gb, err := configtxtest.MakeGenesisBlock(testChannelID)
	assert.NoError(t, err, "Failed to create test configuration block")
	bytes, err := proto.Marshal(gb)
	assert.NoError(t, err)
	cid, err := protoutil.GetChannelIDFromBlockBytes(bytes)
	assert.NoError(t, err)
	assert.Equal(t, testChannelID, cid, "Failed to return expected chain ID")

	// bad block bytes
	_, err = protoutil.GetChannelIDFromBlockBytes([]byte("bad block"))
	assert.Error(t, err, "Expected error with malformed block bytes")
}

func TestGetChannelIDFromBlock(t *testing.T) {
	var err error
	var gb *common.Block
	var cid string

	// nil block
	_, err = protoutil.GetChannelIDFromBlock(gb)
	assert.Error(t, err, "Expected error getting channel id from nil block")

	gb, err = configtxtest.MakeGenesisBlock(testChannelID)
	assert.NoError(t, err, "Failed to create test configuration block")

	cid, err = protoutil.GetChannelIDFromBlock(gb)
	assert.NoError(t, err, "Failed to get chain ID from block")
	assert.Equal(t, testChannelID, cid, "Failed to return expected chain ID")

	// missing data
	badBlock := gb
	badBlock.Data = nil
	_, err = protoutil.GetChannelIDFromBlock(badBlock)
	assert.Error(t, err, "Expected error with missing block data")

	// no envelope
	badBlock = &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{[]byte("bad envelope")},
		},
	}
	_, err = protoutil.GetChannelIDFromBlock(badBlock)
	assert.Error(t, err, "Expected error with no envelope in data")

	// bad payload
	env, _ := proto.Marshal(&cb.Envelope{
		Payload: []byte("bad payload"),
	})
	badBlock = &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{env},
		},
	}
	_, err = protoutil.GetChannelIDFromBlock(badBlock)
	assert.Error(t, err, "Expected error - malformed payload")

	// bad channel header
	payload, _ := proto.Marshal(&cb.Payload{
		Header: &cb.Header{
			ChannelHeader: []byte("bad header"),
		},
	})
	env, _ = proto.Marshal(&cb.Envelope{
		Payload: payload,
	})
	badBlock = &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{env},
		},
	}
	_, err = protoutil.GetChannelIDFromBlock(badBlock)
	assert.Error(t, err, "Expected error with malformed channel header")

	// nil payload header
	payload, _ = proto.Marshal(&cb.Payload{})
	env, _ = proto.Marshal(&cb.Envelope{
		Payload: payload,
	})
	badBlock = &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{env},
		},
	}
	_, err = protoutil.GetChannelIDFromBlock(badBlock)
	assert.Error(t, err, "Expected error when payload header is nil")
}

func TestGetBlockFromBlockBytes(t *testing.T) {
	testChainID := "myuniquetestchainid"
	gb, err := configtxtest.MakeGenesisBlock(testChainID)
	assert.NoError(t, err, "Failed to create test configuration block")
	blockBytes, err := protoutil.Marshal(gb)
	assert.NoError(t, err, "Failed to marshal block")
	_, err = protoutil.UnmarshalBlock(blockBytes)
	assert.NoError(t, err, "to get block from block bytes")

	// bad block bytes
	_, err = protoutil.UnmarshalBlock([]byte("bad block"))
	assert.Error(t, err, "Expected error for malformed block bytes")
}

func TestGetMetadataFromBlock(t *testing.T) {
	t.Run("new block", func(t *testing.T) {
		block := protoutil.NewBlock(0, nil)
		md, err := protoutil.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
		assert.NoError(t, err, "Unexpected error extracting metadata from new block")
		assert.Nil(t, md.Value, "Expected metadata field value to be nil")
		assert.Equal(t, 0, len(md.Value), "Expected length of metadata field value to be 0")
		md = protoutil.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_ORDERER)
		assert.NotNil(t, md, "Expected to get metadata from block")
	})
	t.Run("no metadata", func(t *testing.T) {
		block := protoutil.NewBlock(0, nil)
		block.Metadata = nil
		_, err := protoutil.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
		assert.Error(t, err, "Expected error with nil metadata")
		assert.Contains(t, err.Error(), "no metadata in block")
	})
	t.Run("no metadata at index", func(t *testing.T) {
		block := protoutil.NewBlock(0, nil)
		block.Metadata.Metadata = [][]byte{{1, 2, 3}}
		_, err := protoutil.GetMetadataFromBlock(block, cb.BlockMetadataIndex_LAST_CONFIG)
		assert.Error(t, err, "Expected error with nil metadata")
		assert.Contains(t, err.Error(), "no metadata at index")
	})
	t.Run("malformed metadata", func(t *testing.T) {
		block := protoutil.NewBlock(0, nil)
		block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = []byte("bad metadata")
		_, err := protoutil.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
		assert.Error(t, err, "Expected error with malformed metadata")
		assert.Contains(t, err.Error(), "error unmarshaling metadata at index [ORDERER]")
		assert.Panics(t, func() {
			_ = protoutil.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_ORDERER)
		}, "Expected panic with malformed metadata")
	})
}

func TestGetConsenterMetadataFromBlock(t *testing.T) {
	var cases = []struct {
		name       string
		value      []byte
		signatures []byte
		orderer    []byte
		pass       bool
	}{
		{
			name:       "empty",
			value:      nil,
			signatures: nil,
			orderer:    nil,
			pass:       true,
		},
		{
			name:  "signature only",
			value: []byte("hello"),
			signatures: protoutil.MarshalOrPanic(&cb.Metadata{
				Value: protoutil.MarshalOrPanic(&cb.OrdererBlockMetadata{
					ConsenterMetadata: protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("hello")}),
				}),
			}),
			orderer: nil,
			pass:    true,
		},
		{
			name:       "orderer only",
			value:      []byte("hello"),
			signatures: nil,
			orderer:    protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("hello")}),
			pass:       true,
		},
		{
			name:  "both signatures and orderer",
			value: []byte("hello"),
			signatures: protoutil.MarshalOrPanic(&cb.Metadata{
				Value: protoutil.MarshalOrPanic(&cb.OrdererBlockMetadata{
					ConsenterMetadata: protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("hello")}),
				}),
			}),
			orderer: protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("hello")}),
			pass:    true,
		},
		{
			name:       "malformed OrdererBlockMetadata",
			signatures: protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("malformed")}),
			orderer:    nil,
			pass:       false,
		},
	}

	for _, test := range cases {
		block := protoutil.NewBlock(0, nil)
		block.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES] = test.signatures
		block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = test.orderer
		result, err := protoutil.GetConsenterMetadataFromBlock(block)

		if test.pass {
			require.NoError(t, err)
			require.Equal(t, result.Value, test.value)
		} else {
			require.Error(t, err)
		}
	}
}

func TestInitBlockMeta(t *testing.T) {
	// block with no metadata
	block := &cb.Block{}
	protoutil.InitBlockMetadata(block)
	// should have 3 entries
	assert.Equal(t, 5, len(block.Metadata.Metadata), "Expected block to have 5 metadata entries")

	// block with a single entry
	block = &cb.Block{
		Metadata: &cb.BlockMetadata{},
	}
	block.Metadata.Metadata = append(block.Metadata.Metadata, []byte{})
	protoutil.InitBlockMetadata(block)
	// should have 3 entries
	assert.Equal(t, 5, len(block.Metadata.Metadata), "Expected block to have 5 metadata entries")
}

func TestCopyBlockMetadata(t *testing.T) {
	srcBlock := protoutil.NewBlock(0, nil)
	dstBlock := &cb.Block{}

	metadata, _ := proto.Marshal(&cb.Metadata{
		Value: []byte("orderer metadata"),
	})
	srcBlock.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = metadata
	protoutil.CopyBlockMetadata(srcBlock, dstBlock)

	// check that the copy worked
	assert.Equal(t, len(srcBlock.Metadata.Metadata), len(dstBlock.Metadata.Metadata),
		"Expected target block to have same number of metadata entries after copy")
	assert.Equal(t, metadata, dstBlock.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER],
		"Unexpected metadata from target block")
}

func TestGetLastConfigIndexFromBlock(t *testing.T) {
	index := uint64(2)
	block := protoutil.NewBlock(0, nil)

	t.Run("block with last config metadata in signatures field", func(t *testing.T) {
		block.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES] = protoutil.MarshalOrPanic(&cb.Metadata{
			Value: protoutil.MarshalOrPanic(&cb.OrdererBlockMetadata{
				LastConfig: &cb.LastConfig{Index: 2},
			}),
		})
		result, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.NoError(t, err, "Unexpected error returning last config index")
		assert.Equal(t, index, result, "Unexpected last config index returned from block")
		result = protoutil.GetLastConfigIndexFromBlockOrPanic(block)
		assert.Equal(t, index, result, "Unexpected last config index returned from block")
	})

	t.Run("block with malformed signatures", func(t *testing.T) {
		block.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES] = []byte("apple")
		_, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve metadata: error unmarshaling metadata at index [SIGNATURES]")
	})

	t.Run("block with malformed orderer block metadata", func(t *testing.T) {
		block.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES] = protoutil.MarshalOrPanic(&cb.Metadata{Value: []byte("banana")})
		_, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal orderer block metadata")
	})

	// TODO: FAB-15864 remove the tests below when we stop supporting upgrade from
	//       pre-1.4.1 orderer
	t.Run("block with deprecated (pre-1.4.1) last config", func(t *testing.T) {
		block = protoutil.NewBlock(0, nil)
		block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = protoutil.MarshalOrPanic(&cb.Metadata{
			Value: protoutil.MarshalOrPanic(&cb.LastConfig{
				Index: index,
			}),
		})
		result, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.NoError(t, err, "Unexpected error returning last config index")
		assert.Equal(t, index, result, "Unexpected last config index returned from block")
		result = protoutil.GetLastConfigIndexFromBlockOrPanic(block)
		assert.Equal(t, index, result, "Unexpected last config index returned from block")
	})

	t.Run("malformed metadata", func(t *testing.T) {
		block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = []byte("bad metadata")
		_, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve metadata: error unmarshaling metadata at index [LAST_CONFIG]")
	})

	t.Run("malformed last config", func(t *testing.T) {
		block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = protoutil.MarshalOrPanic(&cb.Metadata{
			Value: []byte("bad last config"),
		})
		_, err := protoutil.GetLastConfigIndexFromBlock(block)
		assert.Error(t, err, "Expected error with malformed last config metadata")
		assert.Contains(t, err.Error(), "error unmarshaling LastConfig")
		assert.Panics(t, func() {
			_ = protoutil.GetLastConfigIndexFromBlockOrPanic(block)
		}, "Expected panic with malformed last config metadata")
	})
}

