package database_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoSessionScopedTenantSetting enforces D004 by statically scanning the entire
// codebase (Go source files and SQL migration files) for any attempt to set `app.tenant_id`
// with session scope (e.g. `SET app.tenant_id = ...` or `set_config('app.tenant_id', ..., false)`).
//
// Every tenant binding MUST be transaction-scoped (`is_local = true`) to prevent connection pool leaks.
func TestNoSessionScopedTenantSetting(t *testing.T) {
	root := filepath.Join("..", "..")

	// Pattern 1: `SET app.tenant_id` or `SET SESSION app.tenant_id`
	sessionSetPattern := regexp.MustCompile(`(?i)\bSET\s+(?:SESSION\s+)?app\.tenant_id\b`)

	// Pattern 2: `set_config('app.tenant_id', ..., false)`
	sessionSetConfigPattern := regexp.MustCompile(`(?i)set_config\(\s*'app\.tenant_id'\s*,\s*[^,]+\s*,\s*false\s*\)`)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, git, and hidden directories
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".sql" {
			return nil
		}

		// Skip this test file itself
		if strings.HasSuffix(path, "lint_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read file %s: %v", path, err)
			return nil
		}

		if match := sessionSetPattern.Find(content); match != nil {
			t.Errorf("forbidden session-scoped SET found in %s: %s", path, string(match))
		}

		if match := sessionSetConfigPattern.Find(content); match != nil {
			t.Errorf("forbidden is_local=false set_config found in %s: %s", path, string(match))
		}

		return nil
	})

	if err != nil {
		t.Fatalf("failed walking repository tree: %v", err)
	}
}
