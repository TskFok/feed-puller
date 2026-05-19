package store

import (
	"strings"
	"testing"
)

func TestMigrationsHaveNoForeignKeys(t *testing.T) {
	t.Parallel()
	for i, stmt := range migrations {
		upper := strings.ToUpper(stmt)
		if strings.Contains(upper, "FOREIGN KEY") || strings.Contains(upper, "REFERENCES ") {
			t.Fatalf("migration %d must not define FOREIGN KEY or REFERENCES: found in\n%s", i, stmt)
		}
	}
}
