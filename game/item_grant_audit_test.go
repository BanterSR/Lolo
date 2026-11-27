package game

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type itemGrantAuditDeclaration struct {
	CallCount   int      `json:"callCount"`
	Category    string   `json:"category"`
	Idempotency string   `json:"idempotency"`
	Transaction string   `json:"transaction"`
	Durability  string   `json:"durability"`
}

type itemGrantAuditRegistry struct {
	Schema    int                                 `json:"schema"`
	Functions map[string]itemGrantAuditDeclaration `json:"functions"`
}

var itemGrantAuditCalls = map[string]bool{
	"AddAllTypeItem": true, "AddItemBase": true, "AddItemWeapon": true,
	"AddItemArmor": true, "AddItemPoster": true, "AddItemInscription": true,
	"AddItemPet": true, "AddHead": true, "AddFurnitureItem": true,
	"grantItems": true, "grantRewardItemPool": true, "grantQuestRewards": true,
	"grantProfileBaseRewards": true, "addItemRewards": true, "addDungeonReward": true,
	"mailGrantItem": true, "grantMailAttachments": true, "applyLifeAdditions": true,
}

func loadItemGrantAuditRegistry(t *testing.T) itemGrantAuditRegistry {
	t.Helper()
	data, err := os.ReadFile("item_grant_audit_registry.json")
	if err != nil {
		t.Fatalf("read item grant audit registry: %v", err)
	}
	var registry itemGrantAuditRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("decode item grant audit registry: %v", err)
	}
	return registry
}

func actualItemGrantAuditCalls(t *testing.T) map[string]int {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}
	root := filepath.Clean(filepath.Join(workingDir, "..", ".."))
	gameRoot := workingDir
	counts := map[string]int{}
	fset := token.NewFileSet()
	err = filepath.WalkDir(gameRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		parents := map[ast.Node]ast.Node{}
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if len(stack) > 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			}
			if !itemGrantAuditCalls[name] {
				return true
			}
			function := "<module>"
			for current := parents[call]; current != nil; current = parents[current] {
				if decl, ok := current.(*ast.FuncDecl); ok {
					function = decl.Name.Name
					break
				}
			}
			counts[rel+"::"+function]++
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan item grant call sites: %v", err)
	}
	return counts
}

func TestItemGrantAuditRegistryCoversEveryCallSite(t *testing.T) {
	registry := loadItemGrantAuditRegistry(t)
	actual := actualItemGrantAuditCalls(t)
	for key, count := range actual {
		declaration, ok := registry.Functions[key]
		if !ok {
			t.Errorf("unregistered item-grant function %s; declare idempotency, transaction, and durability", key)
			continue
		}
		if declaration.CallCount != count {
			t.Errorf("item-grant call count changed for %s: actual=%d declared=%d; update its idempotency/transaction declaration", key, count, declaration.CallCount)
		}
	}
	// Zero-call declarations are allowed for indirect control entry points such
	// as ChooseGacha. They must remain explicit but need not own a direct call.
	for key, declaration := range registry.Functions {
		if declaration.CallCount == 0 {
			continue
		}
		if _, ok := actual[key]; !ok {
			t.Errorf("declared item-grant function no longer exists: %s", key)
		}
	}
}

func TestItemGrantAuditDeclarationsAreComplete(t *testing.T) {
	registry := loadItemGrantAuditRegistry(t)
	if registry.Schema != 1 {
		t.Fatalf("unsupported registry schema: %d", registry.Schema)
	}
	keys := make([]string, 0, len(registry.Functions))
	for key := range registry.Functions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		declaration := registry.Functions[key]
		if declaration.Category == "" || declaration.Idempotency == "" || declaration.Transaction == "" || declaration.Durability == "" {
			t.Errorf("incomplete audit declaration for %s: %+v", key, declaration)
		}
	}
}

func TestKnownHighRiskItemGrantPathsRemainDeclared(t *testing.T) {
	registry := loadItemGrantAuditRegistry(t)
	required := []string{
		"Lolo/game/player_character.go::GetCharacterAchievementAward",
		"Lolo/game/player_gacha.go::ChooseGacha",
		"Lolo/game/player_mail.go::claimMail",
		"Lolo/game/player_quest.go::grantQuestRewards",
		"Lolo/game/player_extra_rewards.go::ActivitySandbagReward",
		"Lolo/game/player_extra_rewards.go::ItemSweep",
	}
	for _, key := range required {
		if _, ok := registry.Functions[key]; !ok {
			t.Errorf("high-risk item-grant path missing from audit registry: %s", key)
		}
	}
}
