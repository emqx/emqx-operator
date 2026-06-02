package hocon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

const complianceSpecDir = "compliance"

type complianceCase struct {
	name       string
	path       string
	expect     string
	expectErr  string
	expectJSON string
}

func TestHOCONCompliance(t *testing.T) {
	cases := loadComplianceCases(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			doc, parseErr := ParseDocumentFile(tc.path)
			root, evalErr := doc.Evaluate()
			switch {
			case tc.expectErr != "":
				var actualErr string
				if parseErr != nil {
					actualErr = classifyComplianceParseError(parseErr)
				} else {
					actualErr = classifyComplianceEvalError(evalErr)
				}
				if actualErr == "" {
					t.Fatalf("expected %s error, got success", tc.expectErr)
				}
				if !complianceErrorMatches(tc.expectErr, actualErr) {
					t.Fatalf("expected %s error, got %s", tc.expectErr, actualErr)
				}
			case tc.expect == "ok":
				g.Expect(parseErr).To(Succeed())
				g.Expect(evalErr).To(Succeed())
				actual, err := json.Marshal(root)
				g.Expect(err).To(Succeed())
				g.Expect(actual).To(MatchJSON(tc.expectJSON))
			}
		})
	}
}

func loadComplianceCases(t *testing.T) []complianceCase {
	t.Helper()

	var err error
	var cases []complianceCase
	specFiles1, globErr := fs.Glob(os.DirFS(complianceSpecDir), "*.hocon")
	err = errors.Join(err, globErr)
	specFiles2, globErr := fs.Glob(os.DirFS(complianceSpecDir), "*/*.hocon")
	err = errors.Join(err, globErr)
	for _, path := range slices.Concat(specFiles1, specFiles2) {
		tc, readErr := readComplianceCase(complianceSpecDir, path)
		err = errors.Join(err, readErr)
		if tc != nil {
			cases = append(cases, *tc)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(cases, func(a, b complianceCase) int {
		return strings.Compare(a.name, b.name)
	})
	return cases
}

func readComplianceCase(root, relPath string) (*complianceCase, error) {
	input, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return nil, err
	}

	tc := complianceCase{
		name: filepath.ToSlash(strings.TrimSuffix(relPath, filepath.Ext(relPath))),
		path: filepath.Join(root, relPath),
	}

	for _, line := range strings.Split(string(input), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		header := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if !strings.HasPrefix(header, "@") {
			continue
		}
		switch {
		case strings.HasPrefix(header, "@expect-json "):
			tc.expectJSON = strings.TrimSpace(strings.TrimPrefix(header, "@expect-json "))
		case strings.HasPrefix(header, "@expect "):
			fields := strings.Fields(header)
			if len(fields) < 2 {
				return nil, fmt.Errorf("%s: malformed @expect header", relPath)
			}
			tc.expect = fields[1]
			if tc.expect == "error" {
				if len(fields) != 3 {
					return nil, fmt.Errorf("%s: malformed @expect error header", relPath)
				}
				tc.expectErr = fields[2]
			}
		}
	}
	if tc.expect == "" {
		return nil, nil
	}
	if tc.expect == "ok" && tc.expectJSON == "" {
		return nil, fmt.Errorf("%s: ok case lacks @expect-json", relPath)
	}
	return &tc, nil
}

func classifyComplianceParseError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrMixedPartials) {
		return "concat_error"
	}
	return "scan_error"
}

func complianceErrorMatches(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if expected == "parse_error" && actual == "scan_error" {
		return true
	}
	return false
}

func classifyComplianceEvalError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrBadArrayIndex):
		return "bad_array_index"
	case errors.Is(err, ErrUndefined), errors.Is(err, ErrUnresolvable):
		return "resolve_error"
	case errors.Is(err, ErrMixedPartials):
		return "concat_error"
	case errors.Is(err, ErrIncludeCycle):
		return "cycle"
	case errors.Is(err, ErrIncludeFailed) && errors.Is(err, os.ErrNotExist):
		return "enoent"
	default:
		return "eval_error"
	}
}
