package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"example.com/kube-inspector/internal/check"
)

func TestSampleReportAndNamespace(t *testing.T) {
	for _, test := range []struct {
		name      string
		namespace string
		count     int
		code      int
	}{
		{name: "all", count: 5, code: 2},
		{name: "completed job only", namespace: "batch", count: 0, code: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"--file", "../../testdata/pods.json", "--output", "json", "--namespace", test.namespace}, &stdout, &stderr)
			if code != test.code {
				t.Fatalf("exit=%d want=%d; stderr=%s", code, test.code, stderr.String())
			}
			var got []check.Finding
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil || len(got) != test.count {
				t.Fatalf("report=%s err=%v want_count=%d", stdout.String(), err, test.count)
			}
			if got == nil {
				t.Fatal("JSON report must be an array, not null")
			}
		})
	}
}

func TestExecutionErrorsAreNotHealthyReports(t *testing.T) {
	dir := t.TempDir()
	for _, test := range []struct{ name, content string }{
		{"invalid-json", "{"},
		{"wrong-kind", `{"kind":"List","items":[]}`},
		{"missing-items", `{"kind":"PodList"}`},
		{"missing-identity", `{"kind":"PodList","items":[{}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, test.name+".json")
			if err := os.WriteFile(path, []byte(test.content), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if code := run([]string{"--file", path}, &stdout, &stderr); code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
		})
	}
	for _, args := range [][]string{
		{"--restart-threshold", "0"},
		{"--restart-threshold", "-1"},
		{"--output", "csv"},
		{"--file", filepath.Join(dir, "missing.json")},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("args=%v code=%d", args, code)
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestOutputFailure(t *testing.T) {
	for _, format := range []string{"table", "json"} {
		var stderr bytes.Buffer
		if code := run([]string{"--file", "../../testdata/pods.json", "--output", format}, brokenWriter{}, &stderr); code != 1 {
			t.Fatalf("format=%s exit=%d", format, code)
		}
	}
}
