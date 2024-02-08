package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"

	"github.com/bobg/pretty"
	"golang.org/x/tools/go/packages"
)

func main() {
	var (
		offset   = flag.Int("o", 0, "offset in input, zero-based")
		filename = flag.String("f", "", "input filename")
		showType = flag.Bool("t", false, "show type")
		public   = flag.Bool("tt", false, "show type with public members")
		private  = flag.Bool("A", false, "show type with all members")
	)
	flag.Parse()

	conf := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}
	pp, err := packages.Load(conf, *filename)
	if err != nil {
		log.Fatal(err)
	}

	if len(pp) != 1 {
		log.Fatalf("loaded %d packages, expected 1", len(pp))
	}

	var (
		p = pp[0]
		f = p.Syntax[0]
		v = visitor{pos: f.Pos() + token.Pos(*offset)}
	)
	ast.Walk(&v, f)
	if v.id == nil {
		log.Fatalf("could not find node at position %d", *offset)
	}

	obj := p.TypesInfo.ObjectOf(v.id)
	if obj == nil {
		log.Fatalf("could not resolve identifier %s at position %d", v.id, *offset)
	}
	posn := p.Fset.PositionFor(obj.Pos(), false)
	fmt.Printf("%s:%d:%d\n", posn.Filename, posn.Line, posn.Column)

	fmt.Printf("* obj.Type() is %s (%T)\n", obj.Type(), obj.Type())

	if *public || *private {
		t := obj.Type()
		pretty.WriteTo(t, os.Stdout)
		for named, ok := obj.Type().(*types.Named); ok; t = named.Underlying() {
		}

		switch t := t.(type) {
		case *types.Struct:
			fmt.Println("struct {")
			for i := 0; i < t.NumFields(); i++ {
				v := t.Field(i)
				fmt.Printf("\t%s\n", v)
			}
			fmt.Println("}")

		case *types.Interface:
			fmt.Println("interface {")
			for i := 0; i < t.NumMethods(); i++ {
				m := t.Method(i)
				fmt.Printf("\t%s\n", m)
			}
			fmt.Println("}")

		default:
			fmt.Printf("%s\n", obj.Type())
		}
	} else if *showType {
		fmt.Printf("%s\n", obj.Type())
	}

	// pretty.WriteTo(obj, os.Stdout)
}

type visitor struct {
	pos token.Pos
	id  *ast.Ident
}

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	if id, ok := n.(*ast.Ident); ok && n.Pos() <= v.pos && v.pos < n.End() {
		v.id = id
		return nil
	}
	return v
}
