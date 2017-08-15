package interpreter

type SymbolTable map[string]interface{}

type Scope struct {
	parent  *Scope
	imports FileHash
	symbols SymbolTable
}

func (s *Scope) ResolveSymbol(identifier string) interface{}{
	if entry, ok := s.symbols[identifier]; ok {
		return entry
	}

	if nil != s.parent {
		return s.parent.ResolveSymbol(identifier)
	}

	return nil
}

func (s *Scope) SetSymbol(identifier string, value interface{}) {
	s.symbols[identifier] = value
}

func (s *Scope) Extend() *Scope {
	return ExtendScope(s)
}

func NewRootScope() *Scope {
	return ExtendScope(nil)
}

func ExtendScope(parentScope *Scope) *Scope {
	return &Scope{parent: parentScope, imports: make(FileHash), symbols: make(SymbolTable)}
}
