package depth

import (
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func Test0(t *testing.T) {
	harness.ValidateCase(
		t,
		"test_0",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(0))
		},
	)
}

func Test1(t *testing.T) {
	harness.ValidateCase(
		t,
		"test_1",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(1))
		},
	)
}

func Test2(t *testing.T) {
	harness.ValidateCase(
		t,
		"test_2",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(2))
		},
	)
}

func Test10(t *testing.T) {
	harness.ValidateCase(
		t,
		"test_10",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(10))
		},
	)
}
