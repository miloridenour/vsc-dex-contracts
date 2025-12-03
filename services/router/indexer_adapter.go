package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IndexerPoolQuerier implements PoolQuerier by querying the indexer HTTP API
type IndexerPoolQuerier struct {
	indexerEndpoint string
	httpClient      *http.Client
}

// NewIndexerPoolQuerier creates a new indexer-based pool querier
func NewIndexerPoolQuerier(indexerEndpoint string) *IndexerPoolQuerier {
	return &IndexerPoolQuerier{
		indexerEndpoint: indexerEndpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// IndexerPoolInfo represents pool info from the indexer API
type IndexerPoolInfo struct {
	ID          string  `json:"id"`
	Asset0      string  `json:"asset0"`
	Asset1      string  `json:"asset1"`
	Reserve0    uint64  `json:"reserve0"`
	Reserve1    uint64  `json:"reserve1"`
	Fee         uint64  `json:"fee"` // Fee in basis points (uint64)
	TotalSupply uint64  `json:"total_supply"`
}

// indexerPoolResponse represents the raw response from indexer (Fee as float64)
type indexerPoolResponse struct {
	ID          string  `json:"id"`
	Asset0      string  `json:"asset0"`
	Asset1      string  `json:"asset1"`
	Reserve0    uint64  `json:"reserve0"`
	Reserve1    uint64  `json:"reserve1"`
	Fee         float64 `json:"fee"` // Fee as percentage (float64)
	TotalSupply uint64  `json:"total_supply"`
}

// GetPoolByID retrieves a pool by its contract ID
func (q *IndexerPoolQuerier) GetPoolByID(poolID string) (*IndexerPoolInfo, error) {
	url := fmt.Sprintf("%s/api/v1/pools/%s", q.indexerEndpoint, poolID)
	
	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexer returned status %d", resp.StatusCode)
	}

	// Decode the indexer response (Fee as float64 percentage)
	var indexerPool indexerPoolResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexerPool); err != nil {
		return nil, fmt.Errorf("failed to decode pool response: %w", err)
	}

	// Convert to router format (Fee as uint64 basis points)
	return &IndexerPoolInfo{
		ID:          indexerPool.ID,
		Asset0:      indexerPool.Asset0,
		Asset1:      indexerPool.Asset1,
		Reserve0:    indexerPool.Reserve0,
		Reserve1:    indexerPool.Reserve1,
		Fee:         uint64(indexerPool.Fee * 100), // Convert percentage to basis points
		TotalSupply: indexerPool.TotalSupply,
	}, nil
}

// GetPoolsByAsset retrieves all pools containing the specified asset
func (q *IndexerPoolQuerier) GetPoolsByAsset(asset string) ([]IndexerPoolInfo, error) {
	url := fmt.Sprintf("%s/api/v1/pools", q.indexerEndpoint)
	
	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexer returned status %d", resp.StatusCode)
	}

	// Decode the indexer response (Fee as float64 percentage)
	var indexerPools []indexerPoolResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexerPools); err != nil {
		return nil, fmt.Errorf("failed to decode pools response: %w", err)
	}

	// Filter pools that contain the specified asset and convert to router format
	var matchingPools []IndexerPoolInfo
	for _, indexerPool := range indexerPools {
		if indexerPool.Asset0 == asset || indexerPool.Asset1 == asset {
			matchingPools = append(matchingPools, IndexerPoolInfo{
				ID:          indexerPool.ID,
				Asset0:      indexerPool.Asset0,
				Asset1:      indexerPool.Asset1,
				Reserve0:    indexerPool.Reserve0,
				Reserve1:    indexerPool.Reserve1,
				Fee:         uint64(indexerPool.Fee * 100), // Convert percentage to basis points
				TotalSupply: indexerPool.TotalSupply,
			})
		}
	}

	return matchingPools, nil
}

