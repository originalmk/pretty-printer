package preenter

import (
	"fmt"
	"testing"
)

func TestNilPointer(t *testing.T) {
	type X struct {
		A int
		B string
	}

	x := X{
		A: 3,
		B: "ABC",
	}

	type Y struct {
		C *X
		D *X
	}

	y := Y{
		C: &x,
		D: nil,
	}

	pp := DefaultPrettyPrinter()
	result, err := pp.SprintPretty(y)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(result)
}

func TestNoListStructDoubleIndent(t *testing.T) {
	type X struct {
		A int
		B string
	}

	type Y struct {
		C []X
	}

	y := Y{
		C: []X{
			{1, "TEST"},
		},
	}

	pp := DefaultPrettyPrinter()
	result, err := pp.SprintPretty(y)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(result)
}
