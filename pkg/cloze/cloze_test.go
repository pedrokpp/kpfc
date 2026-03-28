package cloze_test

import (
	"reflect"
	"testing"

	"kpp.dev/kpfc/pkg/cloze"
)

func TestParse_Single(t *testing.T) {
	got := cloze.Parse("{{c1::Paris}} is the capital of France")
	want := []cloze.Deletion{{Index: 1, Answer: "Paris"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

func TestParse_WithHint(t *testing.T) {
	got := cloze.Parse("{{c1::Paris::city}}")
	want := []cloze.Deletion{{Index: 1, Answer: "Paris", Hint: "city"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

func TestParse_Multiple(t *testing.T) {
	got := cloze.Parse("{{c1::Paris}} is the capital of {{c2::France}}")
	want := []cloze.Deletion{
		{Index: 1, Answer: "Paris"},
		{Index: 2, Answer: "France"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

func TestParse_SameIndexMultipleRegions(t *testing.T) {
	got := cloze.Parse("{{c1::one}} and {{c1::two}}")
	want := []cloze.Deletion{
		{Index: 1, Answer: "one"},
		{Index: 1, Answer: "two"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

func TestParse_NoCloze(t *testing.T) {
	got := cloze.Parse("no cloze here")
	if len(got) != 0 {
		t.Errorf("Parse = %v, want empty", got)
	}
}

func TestIndices_Basic(t *testing.T) {
	got := cloze.Indices("{{c1::a}} {{c2::b}} {{c1::c}}")
	want := []int{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Indices = %v, want %v", got, want)
	}
}

func TestIndices_NonSequential(t *testing.T) {
	got := cloze.Indices("{{c1::a}} {{c3::b}}")
	want := []int{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Indices = %v, want %v", got, want)
	}
}

func TestIndices_Empty(t *testing.T) {
	got := cloze.Indices("plain text")
	if len(got) != 0 {
		t.Errorf("Indices = %v, want empty", got)
	}
}

func TestRender_ActiveCloze(t *testing.T) {
	front, back := cloze.Render("{{c1::Paris}} is the capital of France", 1)
	if front != "[...] is the capital of France" {
		t.Errorf("front = %q, want \"[...] is the capital of France\"", front)
	}
	if back != "<b>Paris</b> is the capital of France" {
		t.Errorf("back = %q, want \"<b>Paris</b> is the capital of France\"", back)
	}
}

func TestRender_WithHint(t *testing.T) {
	front, _ := cloze.Render("{{c1::Paris::city}}", 1)
	if front != "[city]" {
		t.Errorf("front = %q, want \"[city]\"", front)
	}
}

func TestRender_InactiveClozeShowsAnswer(t *testing.T) {
	front, back := cloze.Render("{{c1::Paris}} is the capital of {{c2::France}}", 1)
	if front != "[...] is the capital of France" {
		t.Errorf("front = %q", front)
	}
	if back != "<b>Paris</b> is the capital of France" {
		t.Errorf("back = %q", back)
	}
}

func TestRender_SameIndexMultipleRegions(t *testing.T) {
	front, back := cloze.Render("{{c1::one}} and {{c1::two}}", 1)
	if front != "[...] and [...]" {
		t.Errorf("front = %q, want \"[...] and [...]\"", front)
	}
	if back != "<b>one</b> and <b>two</b>" {
		t.Errorf("back = %q, want \"<b>one</b> and <b>two</b>\"", back)
	}
}

func TestRender_NonSequentialIndex(t *testing.T) {
	// c1 and c3, rendering for c3
	front, back := cloze.Render("{{c1::alpha}} and {{c3::gamma}}", 3)
	if front != "alpha and [...]" {
		t.Errorf("front = %q, want \"alpha and [...]\"", front)
	}
	if back != "alpha and <b>gamma</b>" {
		t.Errorf("back = %q, want \"alpha and <b>gamma</b>\"", back)
	}
}
