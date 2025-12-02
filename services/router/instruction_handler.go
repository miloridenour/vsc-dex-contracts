package router

import (
	"fmt"

	"github.com/vsc-eco/vsc-dex-mapping/schemas"
)

// InstructionToSwapParams converts a SwapInstruction to SwapParams for routing
// amountIn should be provided from the deposit/transaction amount
func InstructionToSwapParams(instruction *schemas.SwapInstruction, amountIn int64) (*SwapParams, error) {
	if instruction == nil {
		return nil, fmt.Errorf("instruction cannot be nil")
	}

	// Set default slippage to 50 basis points (0.5%) if not provided
	maxSlippage := uint64(50)
	if instruction.SlippageBps != nil {
		maxSlippage = uint64(*instruction.SlippageBps)
	}

	// Set default min amount out to 0 if not provided
	minAmountOut := int64(0)
	if instruction.MinAmountOut != nil {
		minAmountOut = *instruction.MinAmountOut
	}

	// Set beneficiary and ref bps
	beneficiary := ""
	if instruction.Beneficiary != nil {
		beneficiary = *instruction.Beneficiary
	}

	refBps := uint64(0)
	if instruction.RefBps != nil {
		refBps = uint64(*instruction.RefBps)
	}

	return &SwapParams{
		Sender:         instruction.Recipient,
		AmountIn:       amountIn,
		AssetIn:        instruction.AssetIn,
		AssetOut:       instruction.AssetOut,
		MinAmountOut:   minAmountOut,
		MaxSlippage:    maxSlippage,
		MiddleOutRatio: 0, // Default value, can be adjusted based on routing logic
		Beneficiary:    beneficiary,
		RefBps:         refBps,
	}, nil
}

// ParseAndValidateInstruction parses instruction data and validates it against the schema
func ParseAndValidateInstruction(data []byte) (*schemas.SwapInstruction, error) {
	// Parse the instruction
	instruction, err := schemas.ParseFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse instruction: %w", err)
	}

	// Validate against schema
	if err := schemas.ValidateInstructionStruct(instruction); err != nil {
		return nil, fmt.Errorf("instruction validation failed: %w", err)
	}

	return instruction, nil
}

// ParseAndConvertInstruction parses instruction data, validates it, and converts to SwapParams
func ParseAndConvertInstruction(data []byte, amountIn int64) (*SwapParams, error) {
	instruction, err := ParseAndValidateInstruction(data)
	if err != nil {
		return nil, err
	}

	return InstructionToSwapParams(instruction, amountIn)
}
