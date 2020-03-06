// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"github.com/BerithFoundation/berith-chain/common"
	"github.com/BerithFoundation/berith-chain/consensus"
	"github.com/BerithFoundation/berith-chain/core/types"
	"github.com/BerithFoundation/berith-chain/core/vm"
	"github.com/BerithFoundation/berith-chain/params"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hash.
	GetHeader(common.Hash, uint64) *types.Header
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(msg Message, header *types.Header, chain ChainContext, author *common.Address) vm.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	return vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(header, chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int).Set(header.Time),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		GasLimit:    header.GasLimit,
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

/*
	[BERITH]
	NewEVMContext creates a new context for use in the EVM.
*/
func NewEVMContext2(msg Message, header *types.Header, chain ChainContext, author *common.Address, config *params.ChainConfig) vm.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	return vm.Context{
		CanTransfer2: CanTransfer2,
		Transfer:     Transfer,
		GetHash:      GetHashFn(header, chain),
		Origin:       msg.From(),
		Coinbase:     beneficiary,
		BlockNumber:  new(big.Int).Set(header.Number),
		Time:         new(big.Int).Set(header.Time),
		Difficulty:   new(big.Int).Set(header.Difficulty),
		GasLimit:     header.GasLimit,
		GasPrice:     new(big.Int).Set(msg.GasPrice()),
		Config:       config,
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.Header, chain ChainContext) func(n uint64) common.Hash {
	var cache map[uint64]common.Hash

	return func(n uint64) common.Hash {
		// If there's no hash cache yet, make one
		if cache == nil {
			cache = map[uint64]common.Hash{
				ref.Number.Uint64() - 1: ref.ParentHash,
			}
		}
		// Try to fulfill the request from the cache
		if hash, ok := cache[n]; ok {
			return hash
		}
		// Not cached, iterate the blocks and cache the hashes
		for header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); header != nil; header = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1) {
			cache[header.Number.Uint64()-1] = header.ParentHash
			if n == header.Number.Uint64()-1 {
				return header.ParentHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int, base types.JobWallet) bool {
	if base == types.Main {
		return db.GetBalance(addr).Cmp(amount) >= 0
	} else if base == types.Stake {
		return db.GetStakeBalance(addr).Cmp(amount) >= 0
	}

	return false
}

// CanTransfer2 checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer2(db vm.StateDB, addr common.Address, amount *big.Int, base types.JobWallet, config *params.ChainConfig, blockNumber *big.Int, target types.JobWallet, recipient common.Address) bool {
	if base == types.Main {
		return db.GetBalance(addr).Cmp(amount) >= 0
	} else if base == types.Stake {
		return db.GetStakeBalance(addr).Cmp(amount) >= 0
	}

	/*
		[BERITH]
		Stake Balance 한도값 체크
		target, recipient, blockNumber
	*/
	if base == types.Main && target == types.Stake {
		var (
			maximum = config.Bsrr.StakeMaximum
			stakeBalance = db.GetStakeBalance(recipient)
			totalStakingAmount = stakeBalance.Add(stakeBalance, amount)
		)
		return config.IsBIP4(blockNumber) && totalStakingAmount.Cmp(maximum) != 1
	}

	return false
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount, blockNumber *big.Int, base, target types.JobWallet) {
	/*
		[BERITH]
		Tx 를 state에 적용
	*/
	switch base {
	case types.Main:
		if target == types.Main {
			db.SubBalance(sender, amount)
			db.AddBalance(recipient, amount)
		} else if target == types.Stake {
			//베이스 지갑 차감
			db.SubBalance(sender, amount)
			db.AddStakeBalance(recipient, amount, blockNumber)
		}

		break
	case types.Stake:
		if target == types.Main {
			//스테이크 풀시
			db.RemoveStakeBalance(sender)
		}
		break
	}
}
