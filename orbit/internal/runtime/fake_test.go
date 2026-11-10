package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestFakeFullLifecycle(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	id, err := f.Create(ctx, UnitSpec{Name: "web-1", Image: "img", CPUs: 1, MemoryMB: 512})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Start(ctx, id); err != nil {
		t.Fatalf("start: %v", err)
	}
	if st, _ := f.Inspect(ctx, id); st.State != "running" {
		t.Fatalf("want running, got %q", st.State)
	}
	f.Exit(id, 137, true) // OOM crash
	if st, _ := f.Inspect(ctx, id); st.State != "exited" || !st.OOMKilled {
		t.Fatalf("want exited+oom, got %+v", st)
	}
	if err := f.Remove(ctx, id); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if f.Exists(id) {
		t.Error("unit still exists after remove")
	}
	if st, _ := f.Inspect(ctx, id); st.State != "missing" {
		t.Errorf("want missing, got %q", st.State)
	}
}

func TestFakeSpecValidation(t *testing.T) {
	f := NewFake()
	cases := []UnitSpec{
		{Name: "", Image: "i", CPUs: 1, MemoryMB: 1},
		{Name: "x", Image: "", CPUs: 1, MemoryMB: 1},
		{Name: "x", Image: "i", CPUs: 1, MemoryMB: 0}, // no memory overcommit
		{Name: "x", Image: "i", CPUs: 0, MemoryMB: 1},
	}
	for i, spec := range cases {
		if _, err := f.Create(context.Background(), spec); err == nil {
			t.Errorf("case %d: create accepted invalid spec %+v", i, spec)
		}
	}
}

func TestFakeDuplicateCreate(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	spec := UnitSpec{Name: "dup", Image: "i", CPUs: 1, MemoryMB: 64}
	if _, err := f.Create(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Create(ctx, spec); err == nil {
		t.Error("duplicate create accepted")
	}
}

func TestFakeScriptedFailures(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("docker daemon down")

	f := NewFake()
	f.CreateErr = sentinel
	if _, err := f.Create(ctx, UnitSpec{Name: "a", Image: "i", CPUs: 1, MemoryMB: 64}); !errors.Is(err, sentinel) {
		t.Errorf("CreateErr not surfaced: %v", err)
	}

	f2 := NewFake()
	f2.OpsErr = sentinel
	if err := f2.Remove(ctx, "nope"); !errors.Is(err, sentinel) {
		t.Errorf("OpsErr not surfaced on remove: %v", err)
	}
	if _, err := f2.Inspect(ctx, "nope"); !errors.Is(err, sentinel) {
		t.Errorf("OpsErr not surfaced on inspect: %v", err)
	}
}

func TestFakeMissingUnit(t *testing.T) {
	f := NewFake()
	st, err := f.Inspect(context.Background(), "ghost")
	if err != nil || st.State != "missing" {
		t.Errorf("want missing state, got %+v err=%v", st, err)
	}
	if err := f.Start(context.Background(), "ghost"); !errors.Is(err, ErrMissing) {
		t.Errorf("start ghost: %v", err)
	}
}
