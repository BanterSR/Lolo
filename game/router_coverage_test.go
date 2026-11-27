package game

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Every client request declared in protocol/cmd must have an explicit route.
// This turns the runtime "no route for msg" log into a build-time regression.
func TestEveryRequestHasRoute(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Dir(filepath.Dir(file))
	cmdSource, err := os.ReadFile(filepath.Join(root, "protocol", "cmd", "cmd_id.go"))
	if err != nil {
		t.Fatal(err)
	}
	routerSource, err := os.ReadFile(filepath.Join(root, "game", "game_router.go"))
	if err != nil {
		t.Fatal(err)
	}
	requestPattern := regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+Req)\s*=\s*\d+`)
	routePattern := regexp.MustCompile(`cmd\.([A-Za-z0-9_]+Req)\s*:`)
	casePattern := regexp.MustCompile(`case cmd\.([A-Za-z0-9_]+Req)\s*:`)
	routed := map[string]bool{}
	for _, match := range routePattern.FindAllStringSubmatch(string(routerSource), -1) {
		routed[match[1]] = true
	}
	for _, match := range casePattern.FindAllStringSubmatch(string(routerSource), -1) {
		routed[match[1]] = true
	}
	var missing []string
	for _, match := range requestPattern.FindAllStringSubmatch(string(cmdSource), -1) {
		if !routed[match[1]] {
			missing = append(missing, match[1])
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("%d request(s) without a route:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}
