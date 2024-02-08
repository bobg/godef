package main

import (
	"go/ast"
	"go/token"
	"log"
	"reflect"
)

func unroll(node ast.Node) []*ast.Ident {
	switch node := node.(type) {
	case *ast.Ident:
		return []*ast.Ident{node}
	case *ast.SelectorExpr:
		idents := unroll(node.X)
		return append(idents, node.Sel)
	default:
		log.Fatalf("expr is %T, not identifier", node)
	}
	return nil // not reached
}

func findNode(f *ast.File, pos int) ast.Node {
	return findNodeHelper(f, token.Pos(pos)+f.Pos())
}

func findNodeHelper(node ast.Node, pos token.Pos) (result ast.Node) {
	if node == nil {
		return nil
	}

	if v := reflect.ValueOf(node); v.IsNil() {
		return nil
	}

	if node.Pos() > pos || node.End() <= pos {
		return nil
	}

	switch node := node.(type) {
	case *ast.File:
		for _, d := range node.Decls {
			if n := findNodeHelper(d, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.GenDecl:
		// xxx

	case *ast.FuncDecl:
		if n := findNodeHelper(node.Recv, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Type, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)

	case *ast.FieldList:
		for _, f := range node.List {
			if n := findNodeHelper(f, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.Field:
		for _, id := range node.Names {
			if n := findNodeHelper(id, pos); n != nil {
				return n
			}
		}
		return findNodeHelper(node.Type, pos)

	case *ast.Ident:
		if node.Pos() <= pos && pos < node.End() {
			return node
		}
		return nil

	case *ast.Ellipsis:
		return findNodeHelper(node.Elt, pos)

	case *ast.BasicLit:
		return nil

	case *ast.FuncLit:
		if n := findNodeHelper(node.Type, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)

	case *ast.CompositeLit:
		if n := findNodeHelper(node.Type, pos); n != nil {
			return n
		}
		for _, e := range node.Elts {
			if n := findNodeHelper(e, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.ParenExpr:
		return findNodeHelper(node.X, pos)

	case *ast.SelectorExpr:
		if node.X.Pos() <= pos && pos < node.Sel.End() {
			return node
		}
		return nil

	case *ast.IndexExpr:
		if n := findNodeHelper(node.X, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Index, pos)

	case *ast.SliceExpr:
		if n := findNodeHelper(node.X, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Low, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.High, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Max, pos)

	case *ast.TypeAssertExpr:
		if n := findNodeHelper(node.X, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Type, pos)

	case *ast.CallExpr:
		if n := findNodeHelper(node.Fun, pos); n != nil {
			return n
		}
		for _, a := range node.Args {
			if n := findNodeHelper(a, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.StarExpr:
		return findNodeHelper(node.X, pos)

	case *ast.UnaryExpr:
		return findNodeHelper(node.X, pos)

	case *ast.BinaryExpr:
		if n := findNodeHelper(node.X, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Y, pos)

	case *ast.KeyValueExpr:
		if n := findNodeHelper(node.Key, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Value, pos)

	case *ast.ArrayType:
		if n := findNodeHelper(node.Len, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Elt, pos)

	case *ast.StructType:
		return findNodeHelper(node.Fields, pos)

	case *ast.FuncType:
		if n := findNodeHelper(node.Params, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Results, pos)

	case *ast.InterfaceType:
		return findNodeHelper(node.Methods, pos)

	case *ast.MapType:
		if n := findNodeHelper(node.Key, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Value, pos)

	case *ast.ChanType:
		return findNodeHelper(node.Value, pos)

	case *ast.DeclStmt:
		return findNodeHelper(node.Decl, pos)

	case *ast.LabeledStmt:
		if n := findNodeHelper(node.Label, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Stmt, pos)

	case *ast.ExprStmt:
		return findNodeHelper(node.X, pos)

	case *ast.SendStmt:
		if n := findNodeHelper(node.Chan, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Value, pos)

	case *ast.IncDecStmt:
		return findNodeHelper(node.X, pos)

	case *ast.AssignStmt:
		for _, e := range node.Lhs {
			if n := findNodeHelper(e, pos); n != nil {
				return n
			}
		}
		for _, e := range node.Rhs {
			if n := findNodeHelper(e, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.GoStmt:
		return findNodeHelper(node.Call, pos)

	case *ast.DeferStmt:
		return findNodeHelper(node.Call, pos)

	case *ast.ReturnStmt:
		for _, e := range node.Results {
			if n := findNodeHelper(e, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.BranchStmt:
		return findNodeHelper(node.Label, pos)

	case *ast.BlockStmt:
		for _, s := range node.List {
			if n := findNodeHelper(s, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.IfStmt:
		if n := findNodeHelper(node.Init, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Cond, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Body, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Else, pos)

	case *ast.CaseClause:
		for _, e := range node.List {
			if n := findNodeHelper(e, pos); n != nil {
				return n
			}
		}
		for _, s := range node.Body {
			if n := findNodeHelper(s, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.SwitchStmt:
		if n := findNodeHelper(node.Init, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Tag, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)

	case *ast.TypeSwitchStmt:
		if n := findNodeHelper(node.Init, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Assign, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)

	case *ast.CommClause:
		if n := findNodeHelper(node.Comm, pos); n != nil {
			return n
		}
		for _, s := range node.Body {
			if n := findNodeHelper(s, pos); n != nil {
				return n
			}
		}
		return nil

	case *ast.SelectStmt:
		return findNodeHelper(node.Body, pos)

	case *ast.ForStmt:
		if n := findNodeHelper(node.Init, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Cond, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Post, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)

	case *ast.RangeStmt:
		if n := findNodeHelper(node.Key, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.Value, pos); n != nil {
			return n
		}
		if n := findNodeHelper(node.X, pos); n != nil {
			return n
		}
		return findNodeHelper(node.Body, pos)
	}

	log.Fatalf("node type %T not handled", node)
	return nil // not reached
}
