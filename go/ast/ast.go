// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ast declares the types used to represent syntax trees for Go
// packages.
package ast

import (
	"unicode"
	"unicode/utf8"

	"github.com/bobg/godef/go/token"
)

// ----------------------------------------------------------------------------
// Interfaces
//
// There are 3 main classes of nodes: Expressions and type nodes,
// statement nodes, and declaration nodes. The node names usually
// match the corresponding Go spec production names to which they
// correspond. The node fields correspond to the individual parts
// of the respective productions.
//
// All nodes contain position information marking the beginning of
// the corresponding source text segment; it is accessible via the
// Pos accessor method. Nodes may contain additional position info
// for language constructs where comments may be found between parts
// of the construct (typically any larger, parenthesized subpart).
// That position information is needed to properly position comments
// when printing the construct.

// All node types implement the Node interface.
type Node interface {
	Pos() token.Pos // position of first character belonging to the node
	End() token.Pos // position of first character immediately after the node
}

// All expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
}

// All statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// All declaration nodes implement the Decl interface.
type Decl interface {
	Node
	declNode()
}

// ----------------------------------------------------------------------------
// Comments

// A Comment node represents a single //-style or /*-style comment.
type Comment struct {
	Slash token.Pos // position of "/" starting the comment
	Text  string    // comment text (excluding '\n' for //-style comments)
}

func (c *Comment) Pos() token.Pos { return c.Slash }
func (c *Comment) End() token.Pos { return token.Pos(int(c.Slash) + len(c.Text)) }

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Pos() token.Pos { return g.List[0].Pos() }
func (g *CommentGroup) End() token.Pos { return g.List[len(g.List)-1].End() }

// ----------------------------------------------------------------------------
// Expressions and types

// A Field represents a Field declaration list in a struct type,
// a method list in an interface type, or a parameter/result declaration
// in a signature.
type Field struct {
	Doc     *CommentGroup // associated documentation; or nil
	Names   []*Ident      // field/method/parameter names; or nil if anonymous field
	Type    Expr          // field/method/parameter type
	Tag     *BasicLit     // field tag; or nil
	Comment *CommentGroup // line comments; or nil
}

func (f *Field) Pos() token.Pos {
	if len(f.Names) > 0 {
		return f.Names[0].Pos()
	}
	return f.Type.Pos()
}

func (f *Field) End() token.Pos {
	if f.Tag != nil {
		return f.Tag.End()
	}
	return f.Type.End()
}

// A FieldList represents a list of Fields, enclosed by parentheses or braces.
type FieldList struct {
	Opening token.Pos // position of opening parenthesis/brace, if any
	List    []*Field  // field list; or nil
	Closing token.Pos // position of closing parenthesis/brace, if any
}

func (f *FieldList) Pos() token.Pos {
	if f.Opening.IsValid() {
		return f.Opening
	}
	// the list should not be empty in this case;
	// be conservative and guard against bad ASTs
	if len(f.List) > 0 {
		return f.List[0].Pos()
	}
	return token.NoPos
}

func (f *FieldList) End() token.Pos {
	if f.Closing.IsValid() {
		return f.Closing + 1
	}
	// the list should not be empty in this case;
	// be conservative and guard against bad ASTs
	if n := len(f.List); n > 0 {
		return f.List[n-1].End()
	}
	return token.NoPos
}

// NumFields returns the number of (named and anonymous fields) in a FieldList.
func (f *FieldList) NumFields() int {
	n := 0
	if f != nil {
		for _, g := range f.List {
			m := len(g.Names)
			if m == 0 {
				m = 1 // anonymous field
			}
			n += m
		}
	}
	return n
}

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
type (
	// A BadExpr node is a placeholder for expressions containing
	// syntax errors for which no correct expression nodes can be
	// created.
	//
	BadExpr struct {
		From, To token.Pos // position range of bad expression
	}

	// An Ident node represents an identifier.
	Ident struct {
		NamePos token.Pos // identifier position
		Name    string    // identifier name
		Obj     *Object   // denoted object; or nil
	}

	// An Ellipsis node stands for the "..." type in a
	// parameter list or the "..." length in an array type.
	//
	Ellipsis struct {
		Ellipsis token.Pos // position of "..."
		Elt      Expr      // ellipsis element type (parameter lists only); or nil
	}

	// A BasicLit node represents a literal of basic type.
	BasicLit struct {
		ValuePos token.Pos   // literal position
		Kind     token.Token // token.INT, token.FLOAT, token.IMAG, token.CHAR, or token.STRING
		Value    string      // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`
	}

	// A FuncLit node represents a function literal.
	FuncLit struct {
		Type *FuncType  // function type
		Body *BlockStmt // function body
	}

	// A CompositeLit node represents a composite literal.
	CompositeLit struct {
		Type   Expr      // literal type; or nil
		Lbrace token.Pos // position of "{"
		Elts   []Expr    // list of composite elements; or nil
		Rbrace token.Pos // position of "}"
	}

	// A ParenExpr node represents a parenthesized expression.
	ParenExpr struct {
		Lparen token.Pos // position of "("
		X      Expr      // parenthesized expression
		Rparen token.Pos // position of ")"
	}

	// A SelectorExpr node represents an expression followed by a selector.
	SelectorExpr struct {
		X   Expr   // expression
		Sel *Ident // field selector
	}

	// An IndexExpr node represents an expression followed by an index.
	IndexExpr struct {
		X      Expr      // expression
		Lbrack token.Pos // position of "["
		Index  Expr      // index expression
		Rbrack token.Pos // position of "]"
	}

	// A SliceExpr node represents an expression followed by slice indices.
	SliceExpr struct {
		X      Expr      // expression
		Lbrack token.Pos // position of "["
		Low    Expr      // begin of slice range; or nil
		High   Expr      // end of slice range; or nil
		Max    Expr      // maximum capacity of slice; or nil
		Slice3 bool      // true if 3-index slice (2 colons present)
		Rbrack token.Pos // position of "]"
	}

	// A TypeAssertExpr node represents an expression followed by a
	// type assertion.
	//
	TypeAssertExpr struct {
		X    Expr // expression
		Type Expr // asserted type; nil means type switch X.(type)
	}

	// A CallExpr node represents an expression followed by an argument list.
	CallExpr struct {
		Fun      Expr      // function expression
		Lparen   token.Pos // position of "("
		Args     []Expr    // function arguments; or nil
		Ellipsis token.Pos // position of "...", if any
		Rparen   token.Pos // position of ")"
	}

	// A StarExpr node represents an expression of the form "*" Expression.
	// Semantically it could be a unary "*" expression, or a pointer type.
	//
	StarExpr struct {
		Star token.Pos // position of "*"
		X    Expr      // operand
	}

	// A UnaryExpr node represents a unary expression.
	// Unary "*" expressions are represented via StarExpr nodes.
	//
	UnaryExpr struct {
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		X     Expr        // operand
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X     Expr        // left operand
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		Y     Expr        // right operand
	}

	// A KeyValueExpr node represents (key : value) pairs
	// in composite literals.
	//
	KeyValueExpr struct {
		Key   Expr
		Colon token.Pos // position of ":"
		Value Expr
	}
)

// The direction of a channel type is indicated by one
// of the following constants.
type ChanDir int

const (
	SEND ChanDir = 1 << iota
	RECV
)

// A type is represented by a tree consisting of one
// or more of the following type-specific expression
// nodes.
type (
	// An ArrayType node represents an array or slice type.
	ArrayType struct {
		Lbrack token.Pos // position of "["
		Len    Expr      // Ellipsis node for [...]T array types, nil for slice types
		Elt    Expr      // element type
	}

	// A StructType node represents a struct type.
	StructType struct {
		Struct     token.Pos  // position of "struct" keyword
		Fields     *FieldList // list of field declarations
		Incomplete bool       // true if (source) fields are missing in the Fields list
	}

	// Pointer types are represented via StarExpr nodes.

	// A FuncType node represents a function type.
	FuncType struct {
		Func    token.Pos  // position of "func" keyword
		Params  *FieldList // (incoming) parameters; or nil
		Results *FieldList // (outgoing) results; or nil
	}

	// An InterfaceType node represents an interface type.
	InterfaceType struct {
		Interface  token.Pos  // position of "interface" keyword
		Methods    *FieldList // list of methods
		Incomplete bool       // true if (source) methods are missing in the Methods list
	}

	// A MapType node represents a map type.
	MapType struct {
		Map   token.Pos // position of "map" keyword
		Key   Expr
		Value Expr
	}

	// A ChanType node represents a channel type.
	ChanType struct {
		Begin token.Pos // position of "chan" keyword or "<-" (whichever comes first)
		Dir   ChanDir   // channel direction
		Value Expr      // value type
	}
)

// Pos and End implementations for expression/type nodes.
func (x *BadExpr) Pos() token.Pos  { return x.From }
func (x *Ident) Pos() token.Pos    { return x.NamePos }
func (x *Ellipsis) Pos() token.Pos { return x.Ellipsis }
func (x *BasicLit) Pos() token.Pos { return x.ValuePos }
func (x *FuncLit) Pos() token.Pos  { return x.Type.Pos() }
func (x *CompositeLit) Pos() token.Pos {
	if x.Type != nil {
		return x.Type.Pos()
	}
	return x.Lbrace
}
func (x *ParenExpr) Pos() token.Pos      { return x.Lparen }
func (x *SelectorExpr) Pos() token.Pos   { return x.X.Pos() }
func (x *IndexExpr) Pos() token.Pos      { return x.X.Pos() }
func (x *SliceExpr) Pos() token.Pos      { return x.X.Pos() }
func (x *TypeAssertExpr) Pos() token.Pos { return x.X.Pos() }
func (x *CallExpr) Pos() token.Pos       { return x.Fun.Pos() }
func (x *StarExpr) Pos() token.Pos       { return x.Star }
func (x *UnaryExpr) Pos() token.Pos      { return x.OpPos }
func (x *BinaryExpr) Pos() token.Pos     { return x.X.Pos() }
func (x *KeyValueExpr) Pos() token.Pos   { return x.Key.Pos() }
func (x *ArrayType) Pos() token.Pos      { return x.Lbrack }
func (x *StructType) Pos() token.Pos     { return x.Struct }
func (x *FuncType) Pos() token.Pos       { return x.Func }
func (x *InterfaceType) Pos() token.Pos  { return x.Interface }
func (x *MapType) Pos() token.Pos        { return x.Map }
func (x *ChanType) Pos() token.Pos       { return x.Begin }

func (x *BadExpr) End() token.Pos { return x.To }
func (x *Ident) End() token.Pos   { return token.Pos(int(x.NamePos) + len(x.Name)) }
func (x *Ellipsis) End() token.Pos {
	if x.Elt != nil {
		return x.Elt.End()
	}
	return x.Ellipsis + 3 // len("...")
}
func (x *BasicLit) End() token.Pos     { return token.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *FuncLit) End() token.Pos      { return x.Body.End() }
func (x *CompositeLit) End() token.Pos { return x.Rbrace + 1 }
func (x *ParenExpr) End() token.Pos    { return x.Rparen + 1 }
func (x *SelectorExpr) End() token.Pos { return x.Sel.End() }
func (x *IndexExpr) End() token.Pos    { return x.Rbrack + 1 }
func (x *SliceExpr) End() token.Pos    { return x.Rbrack + 1 }
func (x *TypeAssertExpr) End() token.Pos {
	if x.Type != nil {
		return x.Type.End()
	}
	return x.X.End()
}
func (x *CallExpr) End() token.Pos     { return x.Rparen + 1 }
func (x *StarExpr) End() token.Pos     { return x.X.End() }
func (x *UnaryExpr) End() token.Pos    { return x.X.End() }
func (x *BinaryExpr) End() token.Pos   { return x.Y.End() }
func (x *KeyValueExpr) End() token.Pos { return x.Value.End() }
func (x *ArrayType) End() token.Pos    { return x.Elt.End() }
func (x *StructType) End() token.Pos   { return x.Fields.End() }
func (x *FuncType) End() token.Pos {
	if x.Results != nil {
		return x.Results.End()
	}
	return x.Params.End()
}
func (x *InterfaceType) End() token.Pos { return x.Methods.End() }
func (x *MapType) End() token.Pos       { return x.Value.End() }
func (x *ChanType) End() token.Pos      { return x.Value.End() }

// exprNode() ensures that only expression/type nodes can be
// assigned to an ExprNode.
func (x *BadExpr) exprNode()        {}
func (x *Ident) exprNode()          {}
func (x *Ellipsis) exprNode()       {}
func (x *BasicLit) exprNode()       {}
func (x *FuncLit) exprNode()        {}
func (x *CompositeLit) exprNode()   {}
func (x *ParenExpr) exprNode()      {}
func (x *SelectorExpr) exprNode()   {}
func (x *IndexExpr) exprNode()      {}
func (x *SliceExpr) exprNode()      {}
func (x *TypeAssertExpr) exprNode() {}
func (x *CallExpr) exprNode()       {}
func (x *StarExpr) exprNode()       {}
func (x *UnaryExpr) exprNode()      {}
func (x *BinaryExpr) exprNode()     {}
func (x *KeyValueExpr) exprNode()   {}

func (x *ArrayType) exprNode()     {}
func (x *StructType) exprNode()    {}
func (x *FuncType) exprNode()      {}
func (x *InterfaceType) exprNode() {}
func (x *MapType) exprNode()       {}
func (x *ChanType) exprNode()      {}

// ----------------------------------------------------------------------------
// Convenience functions for Idents

var noPos token.Pos

// NewIdent creates a new Ident without position.
// Useful for ASTs generated by code other than the Go parser.
func NewIdent(name string) *Ident { return &Ident{noPos, name, nil} }

// IsExported returns whether name is an exported Go symbol
// (i.e., whether it begins with an uppercase letter).
func IsExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

// IsExported returns whether id is an exported Go symbol
// (i.e., whether it begins with an uppercase letter).
func (id *Ident) IsExported() bool { return IsExported(id.Name) }

func (id *Ident) String() string {
	if id != nil {
		return id.Name
	}
	return "<nil>"
}

// ----------------------------------------------------------------------------
// Statements

// A statement is represented by a tree consisting of one
// or more of the following concrete statement nodes.
type (
	// A BadStmt node is a placeholder for statements containing
	// syntax errors for which no correct statement nodes can be
	// created.
	//
	BadStmt struct {
		From, To token.Pos // position range of bad statement
	}

	// A DeclStmt node represents a declaration in a statement list.
	DeclStmt struct {
		Decl Decl
	}

	// An EmptyStmt node represents an empty statement.
	// The "position" of the empty statement is the position
	// of the immediately preceding semicolon.
	//
	EmptyStmt struct {
		Semicolon token.Pos // position of preceding ";"
	}

	// A LabeledStmt node represents a labeled statement.
	LabeledStmt struct {
		Label *Ident
		Colon token.Pos // position of ":"
		Stmt  Stmt
	}

	// An ExprStmt node represents a (stand-alone) expression
	// in a statement list.
	//
	ExprStmt struct {
		X Expr // expression
	}

	// A SendStmt node represents a send statement.
	SendStmt struct {
		Chan  Expr
		Arrow token.Pos // position of "<-"
		Value Expr
	}

	// An IncDecStmt node represents an increment or decrement statement.
	IncDecStmt struct {
		X      Expr
		TokPos token.Pos   // position of Tok
		Tok    token.Token // INC or DEC
	}

	// An AssignStmt node represents an assignment or
	// a short variable declaration.
	//
	AssignStmt struct {
		Lhs    []Expr
		TokPos token.Pos   // position of Tok
		Tok    token.Token // assignment token, DEFINE
		Rhs    []Expr
	}

	// A GoStmt node represents a go statement.
	GoStmt struct {
		Go   token.Pos // position of "go" keyword
		Call *CallExpr
	}

	// A DeferStmt node represents a defer statement.
	DeferStmt struct {
		Defer token.Pos // position of "defer" keyword
		Call  *CallExpr
	}

	// A ReturnStmt node represents a return statement.
	ReturnStmt struct {
		Return  token.Pos // position of "return" keyword
		Results []Expr    // result expressions; or nil
	}

	// A BranchStmt node represents a break, continue, goto,
	// or fallthrough statement.
	//
	BranchStmt struct {
		TokPos token.Pos   // position of Tok
		Tok    token.Token // keyword token (BREAK, CONTINUE, GOTO, FALLTHROUGH)
		Label  *Ident      // label name; or nil
	}

	// A BlockStmt node represents a braced statement list.
	BlockStmt struct {
		Lbrace token.Pos // position of "{"
		List   []Stmt
		Rbrace token.Pos // position of "}"
	}

	// An IfStmt node represents an if statement.
	IfStmt struct {
		If   token.Pos // position of "if" keyword
		Init Stmt      // initialization statement; or nil
		Cond Expr      // condition
		Body *BlockStmt
		Else Stmt // else branch; or nil
	}

	// A CaseClause represents a case of an expression or type switch statement.
	CaseClause struct {
		Case  token.Pos // position of "case" or "default" keyword
		List  []Expr    // list of expressions or types; nil means default case
		Colon token.Pos // position of ":"
		Body  []Stmt    // statement list; or nil
	}

	// A SwitchStmt node represents an expression switch statement.
	SwitchStmt struct {
		Switch token.Pos  // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Tag    Expr       // tag expression; or nil
		Body   *BlockStmt // CaseClauses only
	}

	// An TypeSwitchStmt node represents a type switch statement.
	TypeSwitchStmt struct {
		Switch token.Pos  // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Assign Stmt       // x := y.(type) or y.(type)
		Body   *BlockStmt // CaseClauses only
	}

	// A CommClause node represents a case of a select statement.
	CommClause struct {
		Case  token.Pos // position of "case" or "default" keyword
		Comm  Stmt      // send or receive statement; nil means default case
		Colon token.Pos // position of ":"
		Body  []Stmt    // statement list; or nil
	}

	// An SelectStmt node represents a select statement.
	SelectStmt struct {
		Select token.Pos  // position of "select" keyword
		Body   *BlockStmt // CommClauses only
	}

	// A ForStmt represents a for statement.
	ForStmt struct {
		For  token.Pos // position of "for" keyword
		Init Stmt      // initialization statement; or nil
		Cond Expr      // condition; or nil
		Post Stmt      // post iteration statement; or nil
		Body *BlockStmt
	}

	// A RangeStmt represents a for statement with a range clause.
	RangeStmt struct {
		For        token.Pos   // position of "for" keyword
		Key, Value Expr        // Key, Value may be nil
		TokPos     token.Pos   // position of Tok; invalid if Key == nil
		Tok        token.Token // ILLEGAL if Key == nil, ASSIGN, DEFINE
		X          Expr        // value to range over
		Body       *BlockStmt
	}
)

// Pos and End implementations for statement nodes.
func (s *BadStmt) Pos() token.Pos        { return s.From }
func (s *DeclStmt) Pos() token.Pos       { return s.Decl.Pos() }
func (s *EmptyStmt) Pos() token.Pos      { return s.Semicolon }
func (s *LabeledStmt) Pos() token.Pos    { return s.Label.Pos() }
func (s *ExprStmt) Pos() token.Pos       { return s.X.Pos() }
func (s *SendStmt) Pos() token.Pos       { return s.Chan.Pos() }
func (s *IncDecStmt) Pos() token.Pos     { return s.X.Pos() }
func (s *AssignStmt) Pos() token.Pos     { return s.Lhs[0].Pos() }
func (s *GoStmt) Pos() token.Pos         { return s.Go }
func (s *DeferStmt) Pos() token.Pos      { return s.Defer }
func (s *ReturnStmt) Pos() token.Pos     { return s.Return }
func (s *BranchStmt) Pos() token.Pos     { return s.TokPos }
func (s *BlockStmt) Pos() token.Pos      { return s.Lbrace }
func (s *IfStmt) Pos() token.Pos         { return s.If }
func (s *CaseClause) Pos() token.Pos     { return s.Case }
func (s *SwitchStmt) Pos() token.Pos     { return s.Switch }
func (s *TypeSwitchStmt) Pos() token.Pos { return s.Switch }
func (s *CommClause) Pos() token.Pos     { return s.Case }
func (s *SelectStmt) Pos() token.Pos     { return s.Select }
func (s *ForStmt) Pos() token.Pos        { return s.For }
func (s *RangeStmt) Pos() token.Pos      { return s.For }

func (s *BadStmt) End() token.Pos  { return s.To }
func (s *DeclStmt) End() token.Pos { return s.Decl.End() }
func (s *EmptyStmt) End() token.Pos {
	return s.Semicolon + 1 /* len(";") */
}
func (s *LabeledStmt) End() token.Pos { return s.Stmt.End() }
func (s *ExprStmt) End() token.Pos    { return s.X.End() }
func (s *SendStmt) End() token.Pos    { return s.Value.End() }
func (s *IncDecStmt) End() token.Pos {
	return s.TokPos + 2 /* len("++") */
}
func (s *AssignStmt) End() token.Pos { return s.Rhs[len(s.Rhs)-1].End() }
func (s *GoStmt) End() token.Pos     { return s.Call.End() }
func (s *DeferStmt) End() token.Pos  { return s.Call.End() }
func (s *ReturnStmt) End() token.Pos {
	if n := len(s.Results); n > 0 {
		return s.Results[n-1].End()
	}
	return s.Return + 6 // len("return")
}
func (s *BranchStmt) End() token.Pos {
	if s.Label != nil {
		return s.Label.End()
	}
	return token.Pos(int(s.TokPos) + len(s.Tok.String()))
}
func (s *BlockStmt) End() token.Pos { return s.Rbrace + 1 }
func (s *IfStmt) End() token.Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}
func (s *CaseClause) End() token.Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}
func (s *SwitchStmt) End() token.Pos     { return s.Body.End() }
func (s *TypeSwitchStmt) End() token.Pos { return s.Body.End() }
func (s *CommClause) End() token.Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}
func (s *SelectStmt) End() token.Pos { return s.Body.End() }
func (s *ForStmt) End() token.Pos    { return s.Body.End() }
func (s *RangeStmt) End() token.Pos  { return s.Body.End() }

// stmtNode() ensures that only statement nodes can be
// assigned to a StmtNode.
func (s *BadStmt) stmtNode()        {}
func (s *DeclStmt) stmtNode()       {}
func (s *EmptyStmt) stmtNode()      {}
func (s *LabeledStmt) stmtNode()    {}
func (s *ExprStmt) stmtNode()       {}
func (s *SendStmt) stmtNode()       {}
func (s *IncDecStmt) stmtNode()     {}
func (s *AssignStmt) stmtNode()     {}
func (s *GoStmt) stmtNode()         {}
func (s *DeferStmt) stmtNode()      {}
func (s *ReturnStmt) stmtNode()     {}
func (s *BranchStmt) stmtNode()     {}
func (s *BlockStmt) stmtNode()      {}
func (s *IfStmt) stmtNode()         {}
func (s *CaseClause) stmtNode()     {}
func (s *SwitchStmt) stmtNode()     {}
func (s *TypeSwitchStmt) stmtNode() {}
func (s *CommClause) stmtNode()     {}
func (s *SelectStmt) stmtNode()     {}
func (s *ForStmt) stmtNode()        {}
func (s *RangeStmt) stmtNode()      {}

// ----------------------------------------------------------------------------
// Declarations

// A Spec node represents a single (non-parenthesized) import,
// constant, type, or variable declaration.
type (
	// The Spec type stands for any of *ImportSpec, *ValueSpec, and *TypeSpec.
	Spec interface {
		Node
		specNode()
	}

	// An ImportSpec node represents a single package import.
	ImportSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Name    *Ident        // local package name (including "."); or nil
		Path    *BasicLit     // import path
		Comment *CommentGroup // line comments; or nil
	}

	// A ValueSpec node represents a constant or variable declaration
	// (ConstSpec or VarSpec production).
	//
	ValueSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Names   []*Ident      // value names (len(Names) > 0)
		Type    Expr          // value type; or nil
		Values  []Expr        // initial values; or nil
		Comment *CommentGroup // line comments; or nil
	}

	// A TypeSpec node represents a type declaration (TypeSpec production).
	TypeSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Name    *Ident        // type name
		Assign  token.Pos     // position of '=', if any
		Type    Expr          // *Ident, *ParenExpr, *SelectorExpr, *StarExpr, or any of the *XxxTypes
		Comment *CommentGroup // line comments; or nil
	}
)

// Pos and End implementations for spec nodes.
func (s *ImportSpec) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	return s.Path.Pos()
}
func (s *ValueSpec) Pos() token.Pos { return s.Names[0].Pos() }
func (s *TypeSpec) Pos() token.Pos  { return s.Name.Pos() }

func (s *ImportSpec) End() token.Pos { return s.Path.End() }
func (s *ValueSpec) End() token.Pos {
	if n := len(s.Values); n > 0 {
		return s.Values[n-1].End()
	}
	if s.Type != nil {
		return s.Type.End()
	}
	return s.Names[len(s.Names)-1].End()
}
func (s *TypeSpec) End() token.Pos { return s.Type.End() }

// specNode() ensures that only spec nodes can be
// assigned to a Spec.
func (s *ImportSpec) specNode() {}
func (s *ValueSpec) specNode()  {}
func (s *TypeSpec) specNode()   {}

// A declaration is represented by one of the following declaration nodes.
type (
	// A BadDecl node is a placeholder for declarations containing
	// syntax errors for which no correct declaration nodes can be
	// created.
	//
	BadDecl struct {
		From, To token.Pos // position range of bad declaration
	}

	// A GenDecl node (generic declaration node) represents an import,
	// constant, type or variable declaration. A valid Lparen position
	// (Lparen.Line > 0) indicates a parenthesized declaration.
	//
	// Relationship between Tok value and Specs element type:
	//
	//	token.IMPORT  *ImportSpec
	//	token.CONST   *ValueSpec
	//	token.TYPE    *TypeSpec
	//	token.VAR     *ValueSpec
	//
	GenDecl struct {
		Doc    *CommentGroup // associated documentation; or nil
		TokPos token.Pos     // position of Tok
		Tok    token.Token   // IMPORT, CONST, TYPE, VAR
		Lparen token.Pos     // position of '(', if any
		Specs  []Spec
		Rparen token.Pos // position of ')', if any
	}

	// A FuncDecl node represents a function declaration.
	FuncDecl struct {
		Doc  *CommentGroup // associated documentation; or nil
		Recv *FieldList    // receiver (methods); or nil (functions)
		Name *Ident        // function/method name
		Type *FuncType     // position of Func keyword, parameters and results
		Body *BlockStmt    // function body; or nil (forward declaration)
	}
)

// Pos and End implementations for declaration nodes.
func (d *BadDecl) Pos() token.Pos  { return d.From }
func (d *GenDecl) Pos() token.Pos  { return d.TokPos }
func (d *FuncDecl) Pos() token.Pos { return d.Type.Pos() }

func (d *BadDecl) End() token.Pos { return d.To }
func (d *GenDecl) End() token.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}
func (d *FuncDecl) End() token.Pos {
	if d.Body != nil {
		return d.Body.End()
	}
	return d.Type.End()
}

// declNode() ensures that only declaration nodes can be
// assigned to a DeclNode.
func (d *BadDecl) declNode()  {}
func (d *GenDecl) declNode()  {}
func (d *FuncDecl) declNode() {}

// ----------------------------------------------------------------------------
// Files and packages

// A File node represents a Go source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
type File struct {
	Doc        *CommentGroup   // associated documentation; or nil
	Package    token.Pos       // position of "package" keyword
	Name       *Ident          // package name
	Decls      []Decl          // top-level declarations; or nil
	Scope      *Scope          // package scope (this file only)
	Imports    []*ImportSpec   // imports in this file
	Unresolved []*Ident        // unresolved identifiers in this file
	Comments   []*CommentGroup // list of all comments in the source file
}

func (f *File) Pos() token.Pos { return f.Package }
func (f *File) End() token.Pos {
	if n := len(f.Decls); n > 0 {
		return f.Decls[n-1].End()
	}
	return f.Name.End()
}

// A Package node represents a set of source files
// collectively building a Go package.
type Package struct {
	Name    string             // package name
	Scope   *Scope             // package scope across all files
	Imports map[string]*Object // map of package id -> package object
	Files   map[string]*File   // Go source files by filename
}

func (p *Package) Pos() token.Pos { return token.NoPos }
func (p *Package) End() token.Pos { return token.NoPos }
