package schemas

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseFromJSON parses a SwapInstruction from JSON bytes
func ParseFromJSON(data []byte) (*SwapInstruction, error) {
	var instruction SwapInstruction
	if err := json.Unmarshal(data, &instruction); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if err := instruction.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &instruction, nil
}

// ParseFromQueryParams parses a SwapInstruction from URL query parameters
func ParseFromQueryParams(query string) (*SwapInstruction, error) {
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query string: %w", err)
	}

	instruction := &SwapInstruction{}

	// Parse required fields
	instruction.InstructionType = values.Get("type")
	instruction.SchemaVersion = values.Get("version")
	instruction.AssetIn = values.Get("asset_in")
	instruction.AssetOut = values.Get("asset_out")
	instruction.Recipient = values.Get("recipient")

	// Parse optional fields
	if slippageStr := values.Get("slippage_bps"); slippageStr != "" {
		if slippage, err := strconv.Atoi(slippageStr); err == nil {
			instruction.SlippageBps = &slippage
		}
	}

	if minAmountStr := values.Get("min_amount_out"); minAmountStr != "" {
		if minAmount, err := strconv.ParseInt(minAmountStr, 10, 64); err == nil {
			instruction.MinAmountOut = &minAmount
		}
	}

	if beneficiary := values.Get("beneficiary"); beneficiary != "" {
		instruction.Beneficiary = &beneficiary
	}

	if refBpsStr := values.Get("ref_bps"); refBpsStr != "" {
		if refBps, err := strconv.Atoi(refBpsStr); err == nil {
			instruction.RefBps = &refBps
		}
	}

	if chain := values.Get("return_address.chain"); chain != "" {
		if address := values.Get("return_address.address"); address != "" {
			instruction.ReturnAddr = &ReturnAddress{
				Chain:   chain,
				Address: address,
			}
		}
	}

	// Parse metadata (if present as JSON string)
	if metadataStr := values.Get("metadata"); metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			instruction.Metadata = metadata
		}
	}

	if err := instruction.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return instruction, nil
}

// ParseFromMemo parses a SwapInstruction from a memo string
// It first tries to parse as JSON, then falls back to URL query parameters
func ParseFromMemo(memo string) (*SwapInstruction, error) {
	memo = strings.TrimSpace(memo)

	// Try parsing as JSON first
	if strings.HasPrefix(memo, "{") && strings.HasSuffix(memo, "}") {
		return ParseFromJSON([]byte(memo))
	}

	// Fall back to URL query parameters
	return ParseFromQueryParams(memo)
}

// ParseFromCustomJSON parses a SwapInstruction from VSC custom_json operation payload
// The customJSON parameter should be the raw JSON bytes from cj.Json field
func ParseFromCustomJSON(customJSON []byte) (*SwapInstruction, error) {
	return ParseFromJSON(customJSON)
}
