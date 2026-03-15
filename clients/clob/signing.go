package clob

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/polymarket/go-order-utils/pkg/builder"
	orderutils "github.com/polymarket/go-order-utils/pkg/model"
)

// Signer wraps official go-order-utils signing functionality
type Signer struct {
	privateKey     *ecdsa.PrivateKey
	chainID        int64
	exchangeBuilder *builder.ExchangeOrderBuilderImpl
}

// NewSigner creates a new signer
func NewSigner(privateKeyHex string, chainID int64) (*Signer, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	
	// Create official exchange order builder
	exchangeBuilder := builder.NewExchangeOrderBuilderImpl(big.NewInt(chainID), nil)
	
	return &Signer{
		privateKey:      pk,
		chainID:         chainID,
		exchangeBuilder: exchangeBuilder,
	}, nil
}

// GetAddress returns the Ethereum address
func (s *Signer) GetAddress() string {
	return crypto.PubkeyToAddress(s.privateKey.PublicKey).Hex()
}

// SignClobAuth signs ClobAuth for L1 authentication
func (s *Signer) SignClobAuth(nonce, timestamp int64) (string, error) {
	signerAddr := s.GetAddress()

	domain := apitypes.TypedDataDomain{
		Name:    "ClobAuthDomain",
		Version: "1",
		ChainId: math.NewHexOrDecimal256(s.chainID),
	}

	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
		},
		"ClobAuth": {
			{Name: "address", Type: "address"},
			{Name: "timestamp", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "message", Type: "string"},
		},
	}

	message := map[string]interface{}{
		"address":   signerAddr,
		"timestamp": fmt.Sprintf("%d", timestamp),
		"nonce":     big.NewInt(nonce),
		"message":   "This message attests that I control the given wallet",
	}

	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "ClobAuth",
		Domain:      domain,
		Message:     message,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return "", fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return "", fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256Hash(rawData)

	sig, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	
	// Adjust V value
	sig[64] += 27

	return "0x" + common.Bytes2Hex(sig), nil
}

// SignOrder signs an order using official go-order-utils ExchangeOrderBuilder
func (s *Signer) SignOrder(orderData *orderutils.OrderData, contract orderutils.VerifyingContract) (*orderutils.SignedOrder, error) {
	// Use official builder to build and sign the order
	return s.exchangeBuilder.BuildSignedOrder(s.privateKey, orderData, contract)
}
