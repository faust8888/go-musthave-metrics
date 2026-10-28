package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_ExampleStruct(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "demo")
	if err := os.Mkdir(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	src := `package demo

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "struct.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := generate(root); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(pkgDir, "reset.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	for _, want := range []string{
		"func (rs *ResetableStruct) Reset()",
		"if rs == nil",
		"rs.i = 0",
		`rs.str = ""`,
		"if rs.strP != nil",
		`(*rs.strP) = ""`,
		"rs.s = rs.s[:0]",
		"clear(rs.m)",
		"if rs.child != nil",
		"rs.child.Reset()",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("generated code missing %q\n%s", want, out)
		}
	}
}

func TestGenerate_ResetBehavior(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "demo")
	if err := os.Mkdir(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package demo

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "struct.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generate(root); err != nil {
		t.Fatal(err)
	}

	testSrc := `package demo

import "testing"

func TestReset(t *testing.T) {
	s := "hello"
	child := &ResetableStruct{i: 9, str: "x", s: []int{7}}
	obj := &ResetableStruct{
		i:     42,
		str:   "abc",
		strP:  &s,
		s:     []int{1, 2, 3},
		m:     map[string]string{"a": "b"},
		child: child,
	}
	capBefore := cap(obj.s)
	obj.Reset()
	if obj.i != 0 || obj.str != "" {
		t.Fatalf("primitives: %+v", obj)
	}
	if s != "" {
		t.Fatalf("pointer target: %q", s)
	}
	if obj.strP == nil {
		t.Fatal("strP must stay non-nil")
	}
	if obj.s == nil || len(obj.s) != 0 || cap(obj.s) != capBefore {
		t.Fatalf("slice: len=%d cap=%d nil=%v", len(obj.s), cap(obj.s), obj.s == nil)
	}
	if obj.m == nil || len(obj.m) != 0 {
		t.Fatalf("map: %#v", obj.m)
	}
	if child.i != 0 || child.str != "" || len(child.s) != 0 {
		t.Fatalf("child: %+v", child)
	}
	var nilObj *ResetableStruct
	nilObj.Reset()
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "reset_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./demo")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestGenerate_SkipsUnmarkedAndTestdata(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "demo")
	hidden := filepath.Join(root, "testdata", "hidden")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	unmarked := "package demo\n\ntype Plain struct{ N int }\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "plain.go"), []byte(unmarked), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hidden, "secret.go"), []byte("package hidden\n\n// generate:reset\ntype Secret struct{ N int }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "reset.gen.go")); !os.IsNotExist(err) {
		t.Fatal("unmarked package must not get reset.gen.go")
	}
	if _, err := os.Stat(filepath.Join(hidden, "reset.gen.go")); !os.IsNotExist(err) {
		t.Fatal("testdata must be skipped")
	}
}
