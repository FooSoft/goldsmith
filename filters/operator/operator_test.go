package operator

import (
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/filters/condition"
	"git.sr.ht/~foosoft/goldsmith/harness"
)

func TestAndFalse(t *testing.T) {
	harness.ValidateCase(
		t,
		"and_false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(And(condition.New(false)))
		},
	)
}

func TestAndFalseTrue(t *testing.T) {
	harness.ValidateCase(
		t,
		"and_false_true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(And(condition.New(false), condition.New(true)))
		},
	)
}

func TestAndTrueFalse(t *testing.T) {
	harness.ValidateCase(
		t,
		"and_true_false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(And(condition.New(true), condition.New(false)))
		},
	)
}

func TestAndTrue(t *testing.T) {
	harness.ValidateCase(
		t,
		"and_true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(And(condition.New(true)))
		},
	)
}

func TestOrFalse(t *testing.T) {
	harness.ValidateCase(
		t,
		"or_false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Or(condition.New(false)))
		},
	)
}

func TestOrFalseTrue(t *testing.T) {
	harness.ValidateCase(
		t,
		"or_false_true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Or(condition.New(false), condition.New(true)))
		},
	)
}

func TestOrTrueFalse(t *testing.T) {
	harness.ValidateCase(
		t,
		"or_true_false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Or(condition.New(true), condition.New(false)))
		},
	)
}

func TestOrTrue(t *testing.T) {
	harness.ValidateCase(
		t,
		"or_true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Or(condition.New(true)))
		},
	)
}

func TestNotFalse(t *testing.T) {
	harness.ValidateCase(
		t,
		"not_false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Not(condition.New(false)))
		},
	)
}

func TestNotTrue(t *testing.T) {
	harness.ValidateCase(
		t,
		"not_true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(Not(condition.New(true)))
		},
	)
}
