//go:build ignore

package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

func main() {
	csvFile := "../cmdid.csv"
	outFile := "cmd_id.go"

	f, err := os.Open(csvFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开CSV失败: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取CSV失败: %v\n", err)
		os.Exit(1)
	}

	var names []string
	nameToID := make(map[string]string)
	for _, row := range rows {
		if len(row) == 2 {
			names = append(names, row[0])
			nameToID[row[0]] = row[1]
		}
	}

	var b strings.Builder
	b.WriteString("package cmd\n\nimport (\n\t\"gucooing/lolo/protocol/proto\"\n)\n\nconst (\n")
	for _, name := range names {
		fmt.Fprintf(&b, "\t%-45s = %s\n", name, nameToID[name])
	}
	b.WriteString(")\n\nfunc (c *CmdProtoMap) registerAllMessage() {\n")
	for _, name := range names {
		fmt.Fprintf(&b, "\tc.regMsg(%s, func() any { return new(proto.%s) })\n", name, name)
	}
	b.WriteString("}\n")

	if err := os.WriteFile(outFile, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 %s\n", outFile)
}
