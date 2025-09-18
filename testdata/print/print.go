package print

import (
	"github.com/rogpeppe/godef/a" //@mark(PrintImportDir, "rogpeppe")
	"github.com/rogpeppe/godef/b"
)

type localStruct struct {
	Exported bool
	private  bool
}

func printing() {
start:
	var thing localStruct
	if thing.private {
		thing.Exported = false
		goto start //@mark(PrintStart, "start")
	}
	a.Stuff()    //@mark(PrintA, "a"),mark(PrintStuff, "Stuff")
	var _ = b.S1 //@mark(PrintS1, "S1")
	const c1 = 5
	if c1 == 2 { //@mark(PrintC1, "c1")
	}

	/*@
	 */
}
