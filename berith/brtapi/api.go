package brtapi

import (
	"bitbucket.org/ibizsoftware/berith-chain/core/state"
	"bitbucket.org/ibizsoftware/berith-chain/miner"
	"bitbucket.org/ibizsoftware/berith-chain/rpc"
	"bytes"
	"context"
	"errors"
	// "fmt"
	"math/big"

	"bitbucket.org/ibizsoftware/berith-chain/accounts"
	"bitbucket.org/ibizsoftware/berith-chain/common"
	"bitbucket.org/ibizsoftware/berith-chain/common/hexutil"
	"bitbucket.org/ibizsoftware/berith-chain/core/types"
	"bitbucket.org/ibizsoftware/berith-chain/crypto"
	"bitbucket.org/ibizsoftware/berith-chain/log"

	// "bitbucket.org/ibizsoftware/berith-chain/berith/stake"
)

//PrivateBerithAPI struct of berith private apis
type PrivateBerithAPI struct {
	backend        Backend
	miner		   *miner.Miner
	nonceLock *AddrLocker
	accountManager *accounts.Manager
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
	base  types.JobWallet `json:"base"`
	target  types.JobWallet `json:"target"`
}

// setDefaults is a helper function that fills in default values for unspecified tx fields.
func (args *SendTxArgs) setDefaults(ctx context.Context, b Backend) error {
	if args.Gas == nil {
		args.Gas = new(hexutil.Uint64)
		*(*uint64)(args.Gas) = 90000
	}
	if args.GasPrice == nil {
		price, err := b.SuggestPrice(ctx)
		if err != nil {
			return err
		}
		args.GasPrice = (*hexutil.Big)(price)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.From)
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`Both "data" and "input" are set and not equal. Please use "input" to pass transaction call data.`)
	}
	if args.To == nil {
		// Contract creation
		var input []byte
		if args.Data != nil {
			input = *args.Data
		} else if args.Input != nil {
			input = *args.Input
		}
		if len(input) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
	}
	return nil
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Data != nil {
		input = *args.Data
	} else if args.Input != nil {
		input = *args.Input
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, args.base, args.target)
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, args.base, args.target)
}

//NewPrivateBerithAPI make new instance of PrivateBerithAPI
func NewPrivateBerithAPI(b Backend, m *miner.Miner,nonceLock *AddrLocker) *PrivateBerithAPI {
	return &PrivateBerithAPI{
		backend:        b,
		miner:			m,
		accountManager: b.AccountManager(),
		nonceLock: nonceLock,
	}
}

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	if tx.To() == nil {
		signer := types.MakeSigner(b.ChainConfig(), b.CurrentBlock().Number())
		from, err := types.Sender(signer, tx)
		if err != nil {
			return common.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "fullhash", tx.Hash().Hex(), "contract", addr.Hex())
	} else {
		log.Info("Submitted transaction", "fullhash", tx.Hash().Hex(), "recipient", tx.To())
	}
	return tx.Hash(), nil
}


type WalletTxArgs struct {
	From     common.Address  `json:"from"`
	Value    *hexutil.Big    `json:"value"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
}

func (s *PrivateBerithAPI) GetRewardBalance(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*hexutil.Big, error) {
	state, _, err := s.backend.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	return (*hexutil.Big)(state.GetRewardBalance(address)), state.Error()
}

// RewardToStake
func (s *PrivateBerithAPI) RewardToStake(ctx context.Context, args WalletTxArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	sendTx := new(SendTxArgs)

	sendTx.From = args.From
	sendTx.To = &args.From
	sendTx.Value = args.Value
	sendTx.base = types.Reward
	sendTx.target = types.Stake
	sendTx.Gas = args.Gas
	sendTx.GasPrice = args.GasPrice

	return s.sendTransaction(ctx, *sendTx)
}

// RewardToStake
func (s *PrivateBerithAPI) RewardToBalance(ctx context.Context, args WalletTxArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	sendTx := new(SendTxArgs)

	sendTx.From = args.From
	sendTx.To = &args.From
	sendTx.Value = args.Value
	sendTx.base = types.Reward
	sendTx.target = types.Main
	sendTx.Gas = args.Gas
	sendTx.GasPrice = args.GasPrice

	return s.sendTransaction(ctx, *sendTx)
}

// SendStaking creates a transaction for user staking
func (s *PrivateBerithAPI) Stake(ctx context.Context, args WalletTxArgs) (common.Hash, error) {
	config := s.backend.ChainConfig()
	if args.Value.ToInt().Cmp(config.Bsrr.StakeLowest) <= -1 {
		minimum := new(big.Int).Div(config.Bsrr.StakeLowest, big.NewInt(1e+18))
		log.Info("The minimum number of stakes is ", minimum)
		return 	common.Hash{}, errors.New("staking balance failed")
	}

	// Look up the wallet containing the requested signer
	sendTx := new(SendTxArgs)

	sendTx.From = args.From
	sendTx.To = &args.From
	sendTx.Value = args.Value
	sendTx.base = types.Main
	sendTx.target = types.Stake
	sendTx.Gas = args.Gas
	sendTx.GasPrice = args.GasPrice
	
	return s.sendTransaction(ctx, *sendTx)
}

// SendStaking creates a transaction for user staking
func (s *PrivateBerithAPI) StopStaking(ctx context.Context, args WalletTxArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	sendTx := new(SendTxArgs)

	sendTx.From = args.From
	sendTx.To = &args.From
	sendTx.Value = new(hexutil.Big)
	sendTx.base = types.Stake
	sendTx.target = types.Main
	sendTx.Gas = args.Gas
	sendTx.GasPrice = args.GasPrice

	return s.sendTransaction(ctx, *sendTx)
}


// private trasaction function
func (s *PrivateBerithAPI) sendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error){
	account := accounts.Account{Address: args.From}

	wallet, err := s.backend.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}
	
	s.nonceLock.LockAddr(args.From)
	defer s.nonceLock.UnlockAddr(args.From)

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.backend); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()
	
	var chainID *big.Int
	if config := s.backend.ChainConfig(); config.IsEIP155(s.backend.CurrentBlock().Number()) {
		chainID = config.ChainID
	}
	signed, err := wallet.SignTx(account, tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(ctx, s.backend, signed)
}


func (s *PrivateBerithAPI) GetStakeBalance(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*hexutil.Big, error) {
	state, _, err := s.backend.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	return (*hexutil.Big)(state.GetStakeBalance(address)), state.Error()
}

func (s *PrivateBerithAPI) GetAccountInfo(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*state.Account, error) {
	state, _, err := s.backend.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	return state.GetAccountInfo(address), state.Error()
}