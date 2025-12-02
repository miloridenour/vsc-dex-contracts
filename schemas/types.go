package schemas

import "encoding/json"

// Instruction represents a generic DEX instruction
type Instruction interface {
	Type() string
	Version() string
	Validate() error
}

// ReturnAddress represents a return address with chain and address
type ReturnAddress struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}

// SwapInstruction represents a swap instruction with snake_case JSON tags
type SwapInstruction struct {
	InstructionType string                 `json:"type"`
	SchemaVersion   string                 `json:"version"`
	AssetIn         string                 `json:"asset_in"`
	AssetOut        string                 `json:"asset_out"`
	Recipient       string                 `json:"recipient"`
	SlippageBps     *int                   `json:"slippage_bps,omitempty"`
	MinAmountOut    *int64                 `json:"min_amount_out,omitempty"`
	Beneficiary     *string                `json:"beneficiary,omitempty"`
	RefBps          *int                   `json:"ref_bps,omitempty"`
	ReturnAddr      *ReturnAddress         `json:"return_address,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Type returns the instruction type
func (s SwapInstruction) Type() string {
	return s.InstructionType
}

// Version returns the instruction version
func (s SwapInstruction) Version() string {
	return s.SchemaVersion
}

// Validate performs basic validation on the instruction
func (s SwapInstruction) Validate() error {
	if s.InstructionType == "" {
		return &ValidationError{Field: "type", Message: "type is required"}
	}
	if s.SchemaVersion == "" {
		return &ValidationError{Field: "version", Message: "version is required"}
	}
	if s.AssetIn == "" {
		return &ValidationError{Field: "asset_in", Message: "asset_in is required"}
	}
	if s.AssetOut == "" {
		return &ValidationError{Field: "asset_out", Message: "asset_out is required"}
	}
	if s.Recipient == "" {
		return &ValidationError{Field: "recipient", Message: "recipient is required"}
	}
	return nil
}

// ToJSON serializes the instruction to JSON bytes
func (s SwapInstruction) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}
