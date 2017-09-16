package interpreter

import (
	"fmt"
	"strconv"
	"strings"
)

func (i *Interpreter) handleDeferredLHS(op string, left DeferredResolver, right interface{}) DeferredResolver {
	return func (scope *Scope) (interface{}, error) {
		if lhs, err :=  left(scope); err != nil {
			return nil, err
		} else {
			return i.ApplyOperator(op, lhs, right, scope, false)
		}
	}
}

func (i *Interpreter) handleDeferredRHS(op string, left interface{}, right DeferredResolver) DeferredResolver {
	return func (scope *Scope) (interface{}, error) {
		if rhs, err :=  right(scope); err != nil {
			return nil, err
		} else {
			return i.ApplyOperator(op, left, rhs, scope, false)
		}
	}
}

func (i *Interpreter) addToInt(op string, lhs int64, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	switch right.(type) {
	case int64:
		rhs := right.(int64)
		if "-" == op {
			return lhs - rhs, nil
		}
		return lhs + rhs, nil
	case float64:
		return i.addToFloat(op, float64(lhs), right, scope, deferred)
	case string:
		if "-" == op {
			return nil, incompatible(op)
		}
		return strconv.FormatInt(lhs, 10) + right.(string), nil
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.addToInt(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) addToFloat(op string, lhs float64, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	switch right.(type) {
	case int64:
		rhs := float64(right.(int64))
		if "-" == op {
			return lhs - rhs, nil
		}
		return lhs + rhs, nil
	case float64:
		rhs := right.(float64)
		if "-" == op {
			return lhs - rhs, nil
		}
		return lhs + rhs, nil
	case string:
		if "-" == op {
			return nil, incompatible(op)
		}
		return strconv.FormatFloat(lhs, 'f', -1, 64) + right.(string), nil
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.addToFloat(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) addToString(op string, lhs string, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	if "-" == op {
		return nil, incompatible(op)
	}
	switch right.(type) {
	case string:
		return lhs + right.(string), nil
	case int64:
		return lhs + strconv.FormatInt(right.(int64), 10), nil
	case float64:
		return lhs + strconv.FormatFloat(right.(float64), 'f', -1, 64), nil
	case bool:
		return lhs + strconv.FormatBool(right.(bool)), nil
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.addToString(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) addToBool(op string, lhs bool, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	if "-" == op {
		return nil, incompatible(op)
	}
	switch right.(type) {
	case string:
		return strconv.FormatBool(lhs) + right.(string), nil
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.addToBool(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) multByInt(op string, lhs int64, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	switch right.(type) {
	case int64:
		rhs := right.(int64)
		if "/" == op {
			return float64(lhs) / float64(rhs), nil
		}
		return lhs * rhs, nil
	case float64:
		return i.multByFloat(op, float64(lhs), right, scope, deferred)
	case string:
		if "/" == op {
			return nil, incompatible(op)
		}
		if lhs < int64(0) {
			return nil, fmt.Errorf("Cannot multiply string by negative number")
		}
		rhs := right.(string)
		r := make([]string, lhs)
		for i := int64(0); i < lhs; i++ {
			r[i] = rhs
		}
		return strings.Join(r, ""), nil
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.multByInt(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) multByFloat(op string, lhs float64, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	switch right.(type) {
	case int64:
		rhs := float64(right.(int64))
		if "/" == op {
			return lhs / rhs, nil
		}
		return lhs * rhs, nil
	case float64:
		rhs := right.(float64)
		if "/" == op {
			return lhs / rhs, nil
		}
		return lhs * rhs, nil
	case string:
		return i.multByInt(op, int64(lhs), right, scope, deferred)
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.multByFloat(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func (i *Interpreter) multByString(op string, lhs string, right interface{}, scope *Scope, deferred bool) (interface{}, error) {
	switch right.(type) {
	case int64:
		rhs := right.(int64)
		return i.multByInt(op, rhs, lhs, scope, deferred)
	case float64:
		rhs := int64(right.(float64))
		return i.multByInt(op, rhs, lhs, scope, deferred)
	case DeferredResolver:
		if !deferred {
			if rhs, err := right.(DeferredResolver)(scope); err == nil {
				i.multByString(op, lhs, rhs, scope, false)
			} else {
				return nil, err
			}
		}
		return i.handleDeferredRHS(op, lhs, right.(DeferredResolver)), nil
	default:
		return nil, incompatible(op)
	}
}

func incompatible(op string) error {
	return fmt.Errorf("Incompatible types for operator %q", op)
}
