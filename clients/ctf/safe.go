package ctf

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"log"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// Gnosis Safe operation types
const (
	OperationCall         uint8 = 0
	OperationDelegateCall uint8 = 1
)

// SafeWalletClient handles CTF operations via Gnosis Safe execTransaction
type SafeWalletClient struct {
	client       *ethclient.Client
	privateKey   *ecdsa.PrivateKey
	address      common.Address // EOA address (owner of Safe)
	safeAddress  common.Address // Gnosis Safe contract address
	chainID      *big.Int
	safeContract *bind.BoundContract
	safeABI      abi.ABI
	ctfABI       abi.ABI
	erc20ABI     abi.ABI
}

// SafeWalletConfig for SafeWalletClient
type SafeWalletConfig struct {
	PrivateKeyHex string
	SafeAddress   string // Gnosis Safe contract address (= POLY_FUNDER_ADDR)
	RPCURL        string
	ChainID       int64
}

const (
	NEG_RISK_ADAPTER_ADDR = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"
)

const CTF_ABI_FOR_ENCODING = `[
	{"constant":false,"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"partition","type":"uint256[]"},{"name":"amount","type":"uint256"}],"name":"splitPosition","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"partition","type":"uint256[]"},{"name":"amount","type":"uint256"}],"name":"mergePositions","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"indexSets","type":"uint256[]"}],"name":"redeemPositions","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}
]`

const ERC20_ABI_FOR_ENCODING = `[
	{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}
]`

// Gnosis Safe ABI (only the methods we need)
const SAFE_ABI = `[
  {
    "inputs": [
      {"name": "to", "type": "address"},
      {"name": "value", "type": "uint256"},
      {"name": "data", "type": "bytes"},
      {"name": "operation", "type": "uint8"},
      {"name": "safeTxGas", "type": "uint256"},
      {"name": "baseGas", "type": "uint256"},
      {"name": "gasPrice", "type": "uint256"},
      {"name": "gasToken", "type": "address"},
      {"name": "refundReceiver", "type": "address"},
      {"name": "signatures", "type": "bytes"}
    ],
    "name": "execTransaction",
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "inputs": [
      {"name": "to", "type": "address"},
      {"name": "value", "type": "uint256"},
      {"name": "data", "type": "bytes"},
      {"name": "operation", "type": "uint8"},
      {"name": "safeTxGas", "type": "uint256"},
      {"name": "baseGas", "type": "uint256"},
      {"name": "gasPrice", "type": "uint256"},
      {"name": "gasToken", "type": "address"},
      {"name": "refundReceiver", "type": "address"},
      {"name": "_nonce", "type": "uint256"}
    ],
    "name": "getTransactionHash",
    "outputs": [{"name": "", "type": "bytes32"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "nonce",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "spender", "type": "address"}
    ],
    "name": "allowance",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  }
]`

// NewSafeWalletClient creates a client for CTF operations via Gnosis Safe
func NewSafeWalletClient(ctx context.Context, config SafeWalletConfig) (*SafeWalletClient, error) {
	client, err := ethclient.DialContext(ctx, config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Polygon RPC: %w", err)
	}

	config.PrivateKeyHex = strings.TrimPrefix(config.PrivateKeyHex, "0x")
	pk, err := crypto.HexToECDSA(config.PrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	address := crypto.PubkeyToAddress(pk.PublicKey)
	chainID := big.NewInt(config.ChainID)

	// Parse ABIs
	safeABI, err := abi.JSON(strings.NewReader(SAFE_ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Safe ABI: %w", err)
	}

	ctfABI, err := abi.JSON(strings.NewReader(CTF_ABI_FOR_ENCODING))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CTF ABI: %w", err)
	}

	erc20ABI, err := abi.JSON(strings.NewReader(ERC20_ABI_FOR_ENCODING))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	safeAddr := common.HexToAddress(config.SafeAddress)
	safeContract := bind.NewBoundContract(safeAddr, safeABI, client, client, client)

	return &SafeWalletClient{
		client:       client,
		privateKey:   pk,
		address:      address,
		safeAddress:  safeAddr,
		chainID:      chainID,
		safeContract: safeContract,
		safeABI:      safeABI,
		ctfABI:       ctfABI,
		erc20ABI:     erc20ABI,
	}, nil
}

// Split executes Split operation via Gnosis Safe
func (s *SafeWalletClient) Split(ctx context.Context, conditionID string, amount string, negRisk bool) (string, error) {
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}
	amountWei := amountDec.Shift(USDC_DECIMALS).BigInt()

	// Determine target contract
	targetAddr := common.HexToAddress(CTFContractAddress)
	if negRisk {
		targetAddr = common.HexToAddress(NEG_RISK_ADAPTER_ADDR)
	}

	// 1. Ensure USDC allowance for CTF contract
	if err := s.ensureAllowance(ctx, targetAddr, amountWei); err != nil {
		return "", fmt.Errorf("failed to ensure allowance: %w", err)
	}

	// 2. Encode split operation
	conditionIDBytes := common.HexToHash(conditionID)
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}
	data, err := s.ctfABI.Pack("splitPosition",
		common.HexToAddress(USDCContractAddress),
		common.Hash{}, // parentCollectionId = 0
		conditionIDBytes,
		partition,
		amountWei,
	)
	if err != nil {
		return "", fmt.Errorf("failed to encode split: %w", err)
	}

	// 3. Execute via Safe
	tx, err := s.execSafeTransaction(ctx, targetAddr, big.NewInt(0), data, OperationCall)
	if err != nil {
		return "", fmt.Errorf("failed to execute safe split: %w", err)
	}

	log.Println("INFO:", "[SafeWallet] Split tx submitted", "tx_hash", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}

// Merge executes Merge operation via Gnosis Safe
func (s *SafeWalletClient) Merge(ctx context.Context, conditionID string, amount string, negRisk bool) (string, error) {
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}
	amountWei := amountDec.Shift(USDC_DECIMALS).BigInt()

	// Determine target contract
	targetAddr := common.HexToAddress(CTFContractAddress)
	if negRisk {
		targetAddr = common.HexToAddress(NEG_RISK_ADAPTER_ADDR)
	}

	// Encode merge operation
	conditionIDBytes := common.HexToHash(conditionID)
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}
	data, err := s.ctfABI.Pack("mergePositions",
		common.HexToAddress(USDCContractAddress),
		common.Hash{},
		conditionIDBytes,
		partition,
		amountWei,
	)
	if err != nil {
		return "", fmt.Errorf("failed to encode merge: %w", err)
	}

	tx, err := s.execSafeTransaction(ctx, targetAddr, big.NewInt(0), data, OperationCall)
	if err != nil {
		return "", fmt.Errorf("failed to execute safe merge: %w", err)
	}

	log.Println("INFO:", "[SafeWallet] Merge tx submitted", "tx_hash", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}

// Redeem executes Redeem operation via Gnosis Safe
func (s *SafeWalletClient) Redeem(ctx context.Context, conditionID string, negRisk bool) (string, error) {
	// Determine target contract
	targetAddr := common.HexToAddress(CTFContractAddress)
	if negRisk {
		targetAddr = common.HexToAddress(NEG_RISK_ADAPTER_ADDR)
	}

	// Encode redeem operation
	conditionIDBytes := common.HexToHash(conditionID)
	indexSets := []*big.Int{big.NewInt(1), big.NewInt(2)}
	data, err := s.ctfABI.Pack("redeemPositions",
		common.HexToAddress(USDCContractAddress),
		common.Hash{},
		conditionIDBytes,
		indexSets,
	)
	if err != nil {
		return "", fmt.Errorf("failed to encode redeem: %w", err)
	}

	tx, err := s.execSafeTransaction(ctx, targetAddr, big.NewInt(0), data, OperationCall)
	if err != nil {
		return "", fmt.Errorf("failed to execute safe redeem: %w", err)
	}

	log.Println("INFO:", "[SafeWallet] Redeem tx submitted", "tx_hash", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}

// ensureAllowance checks USDC allowance and approves if insufficient
func (s *SafeWalletClient) ensureAllowance(ctx context.Context, spender common.Address, requiredAmount *big.Int) error {
	// Check current allowance from Safe wallet
	usdcAddr := common.HexToAddress(USDCContractAddress)
	erc20ABI, _ := abi.JSON(strings.NewReader(`[{"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`))
	erc20Contract := bind.NewBoundContract(usdcAddr, erc20ABI, s.client, nil, nil)

	var result []interface{}
	err := erc20Contract.Call(&bind.CallOpts{Context: ctx}, &result, "allowance", s.safeAddress, spender)
	if err != nil {
		return fmt.Errorf("failed to check allowance: %w", err)
	}

	allowance := result[0].(*big.Int)
	log.Println("INFO:", "[SafeWallet] Checking allowance", "current", allowance.String(), "required", requiredAmount.String())

	if allowance.Cmp(requiredAmount) >= 0 {
		log.Println("INFO:", "[SafeWallet] Allowance sufficient, skipping approve")
		return nil
	}

	// Approve MaxUint256 via Safe execTransaction
	log.Println("INFO:", "[SafeWallet] Allowance insufficient, approving...")
	maxUint256 := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	approveData, err := s.erc20ABI.Pack("approve", spender, maxUint256)
	if err != nil {
		return fmt.Errorf("failed to encode approve: %w", err)
	}

	tx, err := s.execSafeTransaction(ctx, usdcAddr, big.NewInt(0), approveData, OperationCall)
	if err != nil {
		return fmt.Errorf("failed to execute safe approve: %w", err)
	}

	log.Println("INFO:", "[SafeWallet] Approve tx submitted, waiting for confirmation", "tx_hash", tx.Hash().Hex())

	// Wait for approval to be mined
	receipt, err := bind.WaitMined(ctx, s.client, tx)
	if err != nil {
		return fmt.Errorf("approval transaction failed: %w", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("approval transaction reverted (tx: %s)", tx.Hash().Hex())
	}

	log.Println("INFO:", "[SafeWallet] Approve confirmed", "block", receipt.BlockNumber.Uint64())
	return nil
}

// execSafeTransaction signs and executes a transaction via Gnosis Safe
// This follows the exact pattern from examples/src/safe-helpers/index.ts
func (s *SafeWalletClient) execSafeTransaction(ctx context.Context, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	zero := big.NewInt(0)
	zeroAddr := common.Address{}

	// 1. Get Safe nonce
	var nonceResult []interface{}
	err := s.safeContract.Call(&bind.CallOpts{Context: ctx}, &nonceResult, "nonce")
	if err != nil {
		return nil, fmt.Errorf("failed to get Safe nonce: %w", err)
	}
	nonce := nonceResult[0].(*big.Int)
	log.Println("INFO:", "[SafeWallet] Safe nonce", "nonce", nonce.String())

	// 2. Get transaction hash from Safe contract
	var hashResult []interface{}
	err = s.safeContract.Call(&bind.CallOpts{Context: ctx}, &hashResult, "getTransactionHash",
		to,        // to
		value,     // value
		data,      // data
		operation, // operation (0 = Call)
		zero,      // safeTxGas
		zero,      // baseGas
		zero,      // gasPrice
		zeroAddr,  // gasToken
		zeroAddr,  // refundReceiver
		nonce,     // _nonce
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction hash: %w", err)
	}
	txHash := hashResult[0].([32]byte)
	log.Println("INFO:", "[SafeWallet] Safe tx hash", "hash", fmt.Sprintf("0x%x", txHash))

	// 3. Sign the transaction hash (EIP-191: "\x19Ethereum Signed Message:\n32" + hash)
	sig, err := crypto.Sign(signHash(txHash[:]), s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction hash: %w", err)
	}

	// Adjust v value for Safe signature format
	// Safe expects v = 31 or 32 (v + 4) for eth_sign style signatures
	if sig[64] == 0 || sig[64] == 1 {
		sig[64] += 31
	} else if sig[64] == 27 || sig[64] == 28 {
		sig[64] += 4
	}

	// 4. Execute transaction via Safe
	opts, err := bind.NewKeyedTransactorWithChainID(s.privateKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	opts.Context = ctx
	opts.Value = big.NewInt(0)

	tx, err := s.safeContract.Transact(opts, "execTransaction",
		to,        // to
		value,     // value
		data,      // data
		operation, // operation
		zero,      // safeTxGas
		zero,      // baseGas
		zero,      // gasPrice
		zeroAddr,  // gasToken
		zeroAddr,  // refundReceiver
		sig,       // signatures
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exec Safe transaction: %w", err)
	}

	return tx, nil
}

// signHash prepends the Ethereum signed message prefix and hashes
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
