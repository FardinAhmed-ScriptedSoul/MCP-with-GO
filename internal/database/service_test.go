package database

import "testing"

func TestIsReadQuery(t *testing.T) {
	for _, query := range []string{"SELECT 1", "WITH records AS (SELECT 1) SELECT * FROM records"} {
		if !isReadQuery(query) {
			t.Errorf("expected read query to be allowed: %q", query)
		}
	}
	for _, query := range []string{"", "DELETE FROM users", "SELECT 1; DROP TABLE users", "SELECT 1 -- comment", "SELECT /* comment */ 1"} {
		if isReadQuery(query) {
			t.Errorf("expected query to be rejected: %q", query)
		}
	}
}
