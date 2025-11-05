package router

import (
	"context"

	"github.com/vsc-eco/vsc-dex-mapping/services/router/types"
)

// MappingAdapter is an alias for the types.MappingAdapter interface
type MappingAdapter = types.MappingAdapter

// DepositValidation is an alias for types.DepositValidation
type DepositValidation = types.DepositValidation

// ChainAdapters manages multiple mapping adapters
type ChainAdapters struct {
	adapters map[string]MappingAdapter
}

// NewChainAdapters creates a new chain adapters manager
func NewChainAdapters() *ChainAdapters {
	return &ChainAdapters{
		adapters: make(map[string]MappingAdapter),
	}
}

// RegisterAdapter registers a mapping adapter for a chain
func (ca *ChainAdapters) RegisterAdapter(adapter MappingAdapter) {
	ca.adapters[adapter.Chain()] = adapter
}

// GetAdapter returns the adapter for a specific chain
func (ca *ChainAdapters) GetAdapter(chain string) (MappingAdapter, bool) {
	adapter, exists := ca.adapters[chain]
	return adapter, exists
}

// GetMappedToken returns the mapped token for a chain
func (ca *ChainAdapters) GetMappedToken(chain string) (string, bool) {
	if adapter, exists := ca.adapters[chain]; exists {
		return adapter.GetMappedToken(), true
	}
	return "", false
}

// ListChains returns all registered chains
func (ca *ChainAdapters) ListChains() []string {
	chains := make([]string, 0, len(ca.adapters))
	for chain := range ca.adapters {
		chains = append(chains, chain)
	}
	return chains
}

// ValidateCrossChainDeposit validates a deposit proof using the appropriate adapter
func (ca *ChainAdapters) ValidateCrossChainDeposit(ctx context.Context, chain string, proof []byte) (*DepositValidation, error) {
	adapter, exists := ca.adapters[chain]
	if !exists {
		return nil, &types.ErrUnsupportedChain{Chain: chain}
	}

	return adapter.ValidateDepositProof(ctx, proof)
}

// Service integration methods

// CanRouteToChain checks if the router can route to a specific chain
func (s *Service) CanRouteToChain(chain string) bool {
	_, exists := s.adapters.GetAdapter(chain)
	return exists
}

// GetCrossChainAssets returns all assets that can be routed cross-chain
func (s *Service) GetCrossChainAssets() []string {
	chains := s.adapters.ListChains()
	assets := make([]string, 0, len(chains))

	for _, chain := range chains {
		if token, exists := s.adapters.GetMappedToken(chain); exists {
			assets = append(assets, token)
		}
	}

	return assets
}

// ComputeCrossChainRoute computes a route that involves cross-chain operations
func (s *Service) ComputeCrossChainRoute(ctx context.Context, params SwapParams) (*SwapResult, error) {
	// Check if this involves cross-chain assets
	fromChain := s.getChainForAsset(params.FromAsset)
	toChain := s.getChainForAsset(params.ToAsset)

	if fromChain == "" && toChain == "" {
		// Regular VSC-only swap
		return s.ComputeRoute(ctx, params)
	}

	// Handle cross-chain routing
	return s.computeCrossChainRoute(ctx, params, fromChain, toChain)
}

// Helper methods

func (s *Service) getChainForAsset(asset string) string {
	for _, chain := range s.adapters.ListChains() {
		if token, exists := s.adapters.GetMappedToken(chain); exists && token == asset {
			return chain
		}
	}
	return ""
}

func (s *Service) computeCrossChainRoute(ctx context.Context, params SwapParams, fromChain, toChain string) (*SwapResult, error) {
	// Implementation depends on the specific cross-chain flow
	// For BTC->HBD: BTC deposit -> VSC swap -> HBD withdrawal (if needed)
	// For HBD->BTC: HBD -> BTC withdrawal

	if fromChain != "" && toChain == "" {
		// Cross-chain inflow (e.g., BTC -> HBD)
		return s.computeInflowRoute(ctx, params, fromChain)
	} else if fromChain == "" && toChain != "" {
		// Cross-chain outflow (e.g., HBD -> BTC)
		return s.computeOutflowRoute(ctx, params, toChain)
	} else {
		// Cross-chain to cross-chain (future feature)
		return nil, &types.ErrUnsupportedRoute{FromChain: fromChain, ToChain: toChain}
	}
}

func (s *Service) computeInflowRoute(ctx context.Context, params SwapParams, fromChain string) (*SwapResult, error) {
	// Route: Deposit mapped token -> VSC swap
	// For now, assume the mapped token is already available
	return s.ComputeRoute(ctx, params)
}

func (s *Service) computeOutflowRoute(ctx context.Context, params SwapParams, toChain string) (*SwapResult, error) {
	// Route: VSC swap -> Burn mapped token for withdrawal
	// For now, assume the mapped token is already available
	return s.ComputeRoute(ctx, params)
}

// Errors are defined in the types package
