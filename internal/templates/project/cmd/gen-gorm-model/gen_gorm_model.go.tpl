// Command gen-gorm-model generates GORM model code from the database schema.
//
// Usage:
//
//	go run ./cmd/gen-gorm-model [flags]
package main

import (
{{- if eq .Storage "memory"}}
	"fmt"
{{- else}}
	"log"
	"path/filepath"

	"gorm.io/gen"
{{- end}}
)

func main() {
{{- if eq .Storage "memory"}}
	fmt.Println("gen-gorm-model: memory storage does not require model generation.")
{{- else}}
	generate()
{{- end}}
}
{{- if ne .Storage "memory"}}

func generate() {
	absPath, err := filepath.Abs("../../internal/{{.AppName}}/model")
	if err != nil {
		log.Fatalf("failed to resolve model path: %v", err)
	}

	g := gen.NewGenerator(gen.Config{
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		ModelPkgPath:  absPath,
		WithUnitTest:  true,
		FieldNullable: true,
	})

	// TODO: configure database connection and call g.UseDB(db)
	// TODO: call g.GenerateModel("table_name") for each table
	// TODO: call g.Execute()
	_ = g
	log.Println("TODO: configure gen-gorm-model for {{.Module}}")
}
{{- end}}
