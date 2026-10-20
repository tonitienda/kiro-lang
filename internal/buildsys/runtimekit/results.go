package runtimekit

import "fmt"

type resultKind uint8

const (
	resultOK resultKind = iota + 1
	resultErr
)

func okResult(v value) resultValue  { return resultValue{Kind: resultOK, Value: v} }
func errResult(v value) resultValue { return resultValue{Kind: resultErr, Err: v} }

// This control signal is consumed only at the enclosing user-function boundary.
// Domain failures remain values; runtime/type failures remain ordinary Go errors.
type propagationSignal struct{ Result resultValue }

func (p *propagationSignal) Error() string { return "result propagation" }

func terminalValue(v value) (value, error) {
	if result, ok := v.(resultValue); ok {
		switch result.Kind {
		case resultOK:
			return result.Value, nil
		case resultErr:
			return nil, fmt.Errorf("unhandled Err: %v", toPlain(result.Err))
		default:
			return nil, fmt.Errorf("invalid result tag")
		}
	}
	return v, nil
}
