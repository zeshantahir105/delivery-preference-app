package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/migrate-create <migration_name>")
	}
	name := os.Args[1]
	if name == "" {
		log.Fatal("migration name required")
	}
	// Sanitize: only alphanumeric and underscore
	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, name); !ok {
		log.Fatal("migration name must be alphanumeric or underscore only")
	}

	dir := "migrations"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = filepath.Join("..", "migrations")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}

	next := 1
	re := regexp.MustCompile(`^(\d+)_`)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if len(m) == 2 {
			n, _ := strconv.Atoi(m[1])
			if n >= next {
				next = n + 1
			}
		}
	}

	seq := fmt.Sprintf("%05d", next)
	base := filepath.Join(dir, seq+"_"+name)
	upPath := base + ".up.sql"
	downPath := base + ".down.sql"

	if err := os.WriteFile(upPath, []byte("-- "+seq+" "+name+" up\n"), 0644); err != nil {
		log.Fatalf("create %s: %v", upPath, err)
	}
	if err := os.WriteFile(downPath, []byte("-- "+seq+" "+name+" down\n"), 0644); err != nil {
		log.Fatalf("create %s: %v", downPath, err)
	}
	log.Printf("created %s and %s", upPath, downPath)
}
