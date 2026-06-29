package repository_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSQLCImportsStayInRepositoryLayer(t *testing.T) {
	t.Helper()
	module := "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa"
	targets := []string{
		module + "/internal/http/...",
		module + "/internal/service/...",
		module + "/internal/service/agent/...",
		module + "/internal/platform/mcpclient/...",
		module + "/cmd/...",
	}
	sqlcImport := module + "/internal/repository/sqlc"
	for _, target := range targets {
		output, err := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", target).Output()
		if err != nil {
			t.Fatalf("go list %s: %v", target, err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
			if strings.Contains(parts[1], sqlcImport) {
				t.Fatalf("%s must not import generated sqlc package", parts[0])
			}
		}
	}
}
