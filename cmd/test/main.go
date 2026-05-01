package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	workspace = "temp/test_workspace"
	testDB    = "temp/test_workspace/.tagfile/test.db"
)

type testRunner struct {
	failed int
	passed int
}

func (t *testRunner) run(name string, fn func() error) {
	if err := fn(); err != nil {
		fmt.Printf("  FAIL %s: %v\n", name, err)
		t.failed++
	} else {
		fmt.Printf("  PASS %s\n", name)
		t.passed++
	}
}

func (t *testRunner) section(title string) {
	fmt.Printf("\n%s\n%s\n", title, strings.Repeat("=", len(title)))
}

func main() {
	os.MkdirAll(workspace, 0755)

	tr := &testRunner{}

	tr.section("Unit Tests (Repository)")
	testUnitTests(tr)
	os.Remove(testDB)

	tr.section("Attr & Config Tests")
	testAttrAndConfig(tr)
	os.Remove(testDB)

	tr.section("Integration Tests (File & Search)")
	testIntegration(tr)

	fmt.Printf("\n结果: %d passed, %d failed\n", tr.passed, tr.failed)
	if tr.failed > 0 {
		os.Exit(1)
	}
}
