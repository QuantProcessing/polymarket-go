package ctf

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// Contract Addresses (Polygon Mainnet)
const (
	CTFContractAddress  = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	USDCContractAddress = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174" // USDC.e
	NativeUSDCAddress   = "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"
	NegRiskCTFExchange  = "0xC5d563A36AE78145C45a50134d48A1215220f80a"
	NegRiskAdapter      = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"
	USDC_DECIMALS       = 6
)

// ABIs
const (
	ctfABI = `[
		{"inputs":[{"internalType":"address","name":"collateralToken","type":"address"},{"internalType":"bytes32","name":"parentCollectionId","type":"bytes32"},{"internalType":"bytes32","name":"conditionId","type":"bytes32"},{"internalType":"uint256[]","name":"partition","type":"uint256[]"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"splitPosition","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"collateralToken","type":"address"},{"internalType":"bytes32","name":"parentCollectionId","type":"bytes32"},{"internalType":"bytes32","name":"conditionId","type":"bytes32"},{"internalType":"uint256[]","name":"partition","type":"uint256[]"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"mergePositions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"collateralToken","type":"address"},{"internalType":"bytes32","name":"parentCollectionId","type":"bytes32"},{"internalType":"bytes32","name":"conditionId","type":"bytes32"},{"internalType":"uint256[]","name":"indexSets","type":"uint256[]"}],"name":"redeemPositions","outputs":[],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"uint256","name":"id","type":"uint256"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"bytes32","name":"conditionId","type":"bytes32"},{"internalType":"uint256","name":"outcomeIndex","type":"uint256"}],"name":"payoutNumerators","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"bytes32","name":"conditionId","type":"bytes32"}],"name":"payoutDenominator","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`

	erc20ABI = `[
		{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
	]`
)

// CTFClientConfig holds configuration for CTFClient
type CTFClientConfig struct {
	PrivateKeyHex string
	RPCURL        string
	ChainID       int64
	FunderAddress string // Optional: Proxy/Safe address
}

// CTFClient handles interactions with the Conditional Token Framework
type CTFClient struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    *big.Int

	ctfContract        *bind.BoundContract
	usdcContract       *bind.BoundContract
	nativeUsdcContract *bind.BoundContract

	funderAddress common.Address      // Optional: Address that holds funds (if different from EOA)
	safeClient   *SafeWalletClient    // Optional: For Gnosis Safe operations
	useSafe      bool                 // True if using Safe Wallet (funderAddress != address)
} // struct end

// NewCTFClient creates a new CTFClient
func NewCTFClient(config CTFClientConfig) (*CTFClient, error) {
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	config.PrivateKeyHex = strings.TrimPrefix(config.PrivateKeyHex, "0x")
	pk, err := crypto.HexToECDSA(config.PrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	address := crypto.PubkeyToAddress(pk.PublicKey)
	chainID := big.NewInt(config.ChainID)

	parsedCTFABI, err := abi.JSON(strings.NewReader(ctfABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CTF ABI: %w", err)
	}

	parsedERC20ABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	var funderAddress common.Address
	if config.FunderAddress != "" {
		funderAddress = common.HexToAddress(config.FunderAddress)
	} else {
		funderAddress = address
	}

	// Detect Safe Wallet mode: if funderAddress != EOA, use SafeWalletClient
	useSafe := funderAddress.Hex() != address.Hex()
	var safeClient *SafeWalletClient
	if useSafe {
		safeConfig := SafeWalletConfig{
			PrivateKeyHex: config.PrivateKeyHex,
			SafeAddress:   config.FunderAddress,
			RPCURL:        config.RPCURL,
			ChainID:       config.ChainID,
		}
		var err error
		safeClient, err = NewSafeWalletClient(context.Background(), safeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SafeWalletClient: %w", err)
		}
	}

	return &CTFClient{
		client:        client,
		privateKey:    pk,
		address:       address,
		chainID:       chainID,
		funderAddress: funderAddress,
		safeClient:    safeClient,
		useSafe:       useSafe,
		ctfContract: bind.NewBoundContract(
			common.HexToAddress(CTFContractAddress),
			parsedCTFABI,
			client,
			client,
			client,
		),
		usdcContract: bind.NewBoundContract(
			common.HexToAddress(USDCContractAddress),
			parsedERC20ABI,
			client,
			client,
			client,
		),
		nativeUsdcContract: bind.NewBoundContract(
			common.HexToAddress(NativeUSDCAddress),
			parsedERC20ABI,
			client,
			client,
			client,
		),
	}, nil
}

// GetAddress returns the wallet address (EOA)
func (c *CTFClient) GetAddress() string {
	return c.address.Hex()
}

// GetFunderAddress returns the funder address (Proxy or EOA)
func (c *CTFClient) GetFunderAddress() string {
	return c.funderAddress.Hex()
}

// GetUsdcBalance returns the USDC.e balance of the Funder Address
func (c *CTFClient) GetUsdcBalance(ctx context.Context) (decimal.Decimal, error) {
	var results []interface{}
	err := c.usdcContract.Call(&bind.CallOpts{Context: ctx}, &results, "balanceOf", c.funderAddress)
	if err != nil {
		return decimal.Zero, err
	}
	balance := results[0].(*big.Int)
	return decimal.NewFromBigInt(balance, -USDC_DECIMALS), nil
}

// GetNativeUsdcBalance returns the Native USDC balance of the Funder Address
func (c *CTFClient) GetNativeUsdcBalance(ctx context.Context) (decimal.Decimal, error) {
	var results []interface{}
	err := c.nativeUsdcContract.Call(&bind.CallOpts{Context: ctx}, &results, "balanceOf", c.funderAddress)
	if err != nil {
		return decimal.Zero, err
	}
	balance := results[0].(*big.Int)
	return decimal.NewFromBigInt(balance, -USDC_DECIMALS), nil
}

// CheckReadyForCTF checks if the wallet is ready for CTF operations
func (c *CTFClient) CheckReadyForCTF(ctx context.Context, amountStr string) (bool, string, error) {
	requiredAmount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return false, "", fmt.Errorf("invalid amount: %w", err)
	}

	usdcEBal, err := c.GetUsdcBalance(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get USDC.e balance: %w", err)
	}

	nativeBal, err := c.GetNativeUsdcBalance(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get Native USDC balance: %w", err)
	}

	maticBalBig, err := c.client.BalanceAt(ctx, c.address, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to get MATIC balance: %w", err)
	}
	maticBal := decimal.NewFromBigInt(maticBalBig, -18)

	if maticBal.LessThan(decimal.NewFromFloat(0.01)) {
		return false, fmt.Sprintf("Insufficient MATIC for gas. Have: %s, Need: 0.01", maticBal.String()), nil
	}

	if usdcEBal.LessThan(requiredAmount) {
		if nativeBal.GreaterThanOrEqual(requiredAmount) {
			return false, fmt.Sprintf("You have %s native USDC but only %s USDC.e. Swap required.", nativeBal.String(), usdcEBal.String()), nil
		}
		return false, fmt.Sprintf("Insufficient funds. Have: %s USDC.e, Need: %s", usdcEBal.String(), requiredAmount.String()), nil
	}

	return true, "Ready", nil
}

// getTxOpts creates a new TransactOpts
func (c *CTFClient) getTxOpts() (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return nil, err
	}
	// Let ethclient estimate gas
	opts.GasLimit = 0
	return opts, nil
}

// Split splits USDC into YES/NO tokens
// If Safe Wallet is configured, routes to SafeWalletClient; otherwise executes directly
func (c *CTFClient) Split(ctx context.Context, conditionID string, amountStr string) (string, error) {
	// Route to SafeWalletClient if using Gnosis Safe
	if c.useSafe {
		return c.safeClient.Split(ctx, conditionID, amountStr, false)
	}

	// Direct EOA execution
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}
	amountWei := amount.Mul(decimal.NewFromFloat(1e6)).BigInt()

	// 1. Check Allowance
	var results []interface{}
	err = c.usdcContract.Call(&bind.CallOpts{Context: ctx}, &results, "allowance", c.address, common.HexToAddress(CTFContractAddress))
	if err != nil {
		return "", fmt.Errorf("failed to check allowance: %w", err)
	}
	allowance := results[0].(*big.Int)

	if allowance.Cmp(amountWei) < 0 {
		opts, err := c.getTxOpts()
		if err != nil {
			return "", err
		}
		opts.Context = ctx

		// Approve MaxUint256
		maxUint256 := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
		tx, err := c.usdcContract.Transact(opts, "approve", common.HexToAddress(CTFContractAddress), maxUint256)
		if err != nil {
			return "", fmt.Errorf("failed to approve USDC: %w", err)
		}
		// Wait for approval to be mined (simple wait)
		_, err = bind.WaitMined(ctx, c.client, tx)
		if err != nil {
			return "", fmt.Errorf("approval tx failed: %w", err)
		}
	}

	// 2. Split
	opts, err := c.getTxOpts()
	if err != nil {
		return "", err
	}
	opts.Context = ctx

	conditionIdBytes := common.HexToHash(conditionID)
	parentCollectionId := common.Hash{} // 0x0
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}

	tx, err := c.ctfContract.Transact(opts, "splitPosition",
		common.HexToAddress(USDCContractAddress),
		parentCollectionId,
		conditionIdBytes,
		partition,
		amountWei,
	)
	if err != nil {
		return "", fmt.Errorf("split transaction failed: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// Merge merges YES/NO tokens back to USDC
// If Safe Wallet is configured, routes to SafeWalletClient; otherwise executes directly
func (c *CTFClient) Merge(ctx context.Context, conditionID string, amountStr string) (string, error) {
	// Route to SafeWalletClient if using Gnosis Safe
	if c.useSafe {
		return c.safeClient.Merge(ctx, conditionID, amountStr, false)
	}

	// Direct EOA execution
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}
	amountWei := amount.Mul(decimal.NewFromFloat(1e6)).BigInt()

	opts, err := c.getTxOpts()
	if err != nil {
		return "", err
	}
	opts.Context = ctx

	conditionIdBytes := common.HexToHash(conditionID)
	parentCollectionId := common.Hash{}
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}

	tx, err := c.ctfContract.Transact(opts, "mergePositions",
		common.HexToAddress(USDCContractAddress),
		parentCollectionId,
		conditionIdBytes,
		partition,
		amountWei,
	)
	if err != nil {
		return "", fmt.Errorf("merge transaction failed: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// Redeem redeems winning tokens (Standard CTF)
func (c *CTFClient) Redeem(ctx context.Context, conditionID string, outcome string) (string, error) {
	opts, err := c.getTxOpts()
	if err != nil {
		return "", err
	}
	opts.Context = ctx

	conditionIdBytes := common.HexToHash(conditionID)
	parentCollectionId := common.Hash{}

	// Check if configured for Poly (implied by this client)
	// indexSets: [1] for YES, [2] for NO
	var indexSets []*big.Int
	upperOutcome := strings.ToUpper(outcome)
	if upperOutcome == "YES" {
		indexSets = []*big.Int{big.NewInt(1)}
	} else if upperOutcome == "NO" {
		indexSets = []*big.Int{big.NewInt(2)}
	} else {
		return "", fmt.Errorf("invalid outcome: %s", outcome)
	}

	tx, err := c.ctfContract.Transact(opts, "redeemPositions",
		common.HexToAddress(USDCContractAddress),
		parentCollectionId,
		conditionIdBytes,
		indexSets,
	)
	if err != nil {
		return "", fmt.Errorf("redeem transaction failed: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// RedeemByTokenIds redeems winning tokens using Token IDs (CLOB)
func (c *CTFClient) RedeemByTokenIds(ctx context.Context, conditionID string, tokenIds map[string]string, outcome string) (string, error) {
	// In the future: Check balance using tokenIds before submitting tx to save gas?
	return c.Redeem(ctx, conditionID, outcome)
}

// calculatePositionId calculates the position ID for a given condition and index set
// Position ID = keccak256(parentCollectionId, conditionId, indexSet)
// For Polymarket, parentCollectionId is 0.
func (c *CTFClient) calculatePositionId(conditionID string, indexSet int64) (*big.Int, error) {
	parentCollectionId := common.Hash{}
	conditionIdBytes := common.HexToHash(conditionID)
	indexSetBig := big.NewInt(indexSet)

	// Solidity packing:
	// keccak256(abi.encodePacked(parentCollectionId, conditionId, indexSet))
	// parentCollectionId (32 bytes)
	// conditionId (32 bytes)
	// indexSet (256 bits -> 32 bytes)

	data := make([]byte, 0, 96)
	data = append(data, parentCollectionId.Bytes()...)
	data = append(data, conditionIdBytes.Bytes()...)
	data = append(data, common.LeftPadBytes(indexSetBig.Bytes(), 32)...)

	hash := crypto.Keccak256(data)
	return new(big.Int).SetBytes(hash), nil
}

// PositionBalance holds balance info for a condition
type PositionBalance struct {
	ConditionID   string
	YesBalance    decimal.Decimal
	NoBalance     decimal.Decimal
	YesPositionID string
	NoPositionID  string
}

// GetPositionBalance gets token balances using calculated Position IDs (Standard CTF)
func (c *CTFClient) GetPositionBalance(ctx context.Context, conditionID string) (*PositionBalance, error) {
	yesPosID, err := c.calculatePositionId(conditionID, 1) // YES index set = 1
	if err != nil {
		return nil, err
	}
	noPosID, err := c.calculatePositionId(conditionID, 2) // NO index set = 2
	if err != nil {
		return nil, err
	}

	var resultsYes []interface{}
	err = c.ctfContract.Call(&bind.CallOpts{Context: ctx}, &resultsYes, "balanceOf", c.funderAddress, yesPosID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YES balance: %w", err)
	}
	yesBal := resultsYes[0].(*big.Int)

	var resultsNo []interface{}
	err = c.ctfContract.Call(&bind.CallOpts{Context: ctx}, &resultsNo, "balanceOf", c.funderAddress, noPosID)
	if err != nil {
		return nil, fmt.Errorf("failed to get NO balance: %w", err)
	}
	noBal := resultsNo[0].(*big.Int)

	return &PositionBalance{
		ConditionID:   conditionID,
		YesBalance:    decimal.NewFromBigInt(yesBal, -USDC_DECIMALS),
		NoBalance:     decimal.NewFromBigInt(noBal, -USDC_DECIMALS),
		YesPositionID: yesPosID.String(),
		NoPositionID:  noPosID.String(),
	}, nil
}

// TokenIds holds YES/NO token IDs
type TokenIds struct {
	YesTokenID string
	NoTokenID  string
}

// GetPositionBalanceByTokenIds gets token balances using explicit Token IDs (CLOB)
func (c *CTFClient) GetPositionBalanceByTokenIds(ctx context.Context, conditionID string, tokenIds TokenIds) (*PositionBalance, error) {
	yesTokenID, ok := new(big.Int).SetString(tokenIds.YesTokenID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid YES token ID: %s", tokenIds.YesTokenID)
	}
	noTokenID, ok := new(big.Int).SetString(tokenIds.NoTokenID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid NO token ID: %s", tokenIds.NoTokenID)
	}

	var resultsYes []interface{}
	err := c.ctfContract.Call(&bind.CallOpts{Context: ctx}, &resultsYes, "balanceOf", c.funderAddress, yesTokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YES balance: %w", err)
	}
	yesBal := resultsYes[0].(*big.Int)

	var resultsNo []interface{}
	err = c.ctfContract.Call(&bind.CallOpts{Context: ctx}, &resultsNo, "balanceOf", c.funderAddress, noTokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get NO balance: %w", err)
	}
	noBal := resultsNo[0].(*big.Int)

	return &PositionBalance{
		ConditionID:   conditionID,
		YesBalance:    decimal.NewFromBigInt(yesBal, -USDC_DECIMALS),
		NoBalance:     decimal.NewFromBigInt(noBal, -USDC_DECIMALS),
		YesPositionID: tokenIds.YesTokenID,
		NoPositionID:  tokenIds.NoTokenID,
	}, nil
}
