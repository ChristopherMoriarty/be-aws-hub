package config

import "testing"

func TestPostgresURL(t *testing.T) {
	got := postgresURL("postgres", "5432", "aws", "p@ss", "aws_hub")
	want := "postgres://aws:p%40ss@postgres:5432/aws_hub?sslmode=disable"
	if got != want {
		t.Fatalf("url = %s", got)
	}
}
