package main

import (
	"flag"
	"fmt"

	"GoMetaSync.com/internal/collector"
	"GoMetaSync.com/internal/config"
	"GoMetaSync.com/internal/generator"
	"GoMetaSync.com/internal/snapshot"
)

func main() {
	mode := flag.String("mode", "snapshot", "snapshot | diff | generate")
	cfgPath := flag.String("config", "configs/dev.yml", "config file path")
	oldSnapPath := flag.String("old", "", "old snapshot path (for diff)")
	newSnapPath := flag.String("new", "snapshots/dev-latest.json", "new snapshot output path")
	outDir := flag.String("out", "generated_models", "output dir for generated structs")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		panic(err)
	}

	// Build map[name]dsn
	dbMap := make(map[string]string)
	for _, db := range cfg.Databases {
		dbMap[db.Name] = db.BuildDSN()
	}

	switch *mode {
	case "snapshot":
		snap, err := collector.CollectSnapshot(cfg.Env, dbMap)
		if err != nil {
			panic(err)
		}
		if err := snapshot.SaveSnapshot(*newSnapPath, snap); err != nil {
			panic(err)
		}
		fmt.Println("✅ Snapshot saved:", *newSnapPath)
		return

	case "diff":
		oldSnap, err := snapshot.LoadSnapshot(*oldSnapPath)
		if err != nil {
			panic(err)
		}
		newSnap, err := snapshot.LoadSnapshot(*newSnapPath)
		if err != nil {
			panic(err)
		}
		snapshot.Diff(oldSnap, newSnap)
		return

	case "generate":
		// We generate from a snapshot file (the one you pass via --new)
		snap, err := snapshot.LoadSnapshot(*newSnapPath)
		if err != nil {
			panic(err)
		}
		if err := generator.GenerateStructs(snap, *outDir); err != nil {
			panic(err)
		}
		fmt.Println("✅ Structs generated into:", *outDir)
		return
	}

	fmt.Println("Unknown mode:", *mode)
}
