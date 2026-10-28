// Command jcunpack installs the game data from the Screen Antics floppy.
//
// The floppy stores SCRANTIC.SCR and RESOURCE.001 compressed (as SCRANTIC.SC$
// and RESOURCE.00$). This tool decompresses them, copies RESOURCE.MAP, checks
// the results against known MD5 hashes and writes them to the game data folder.
//
// Usage: jcunpack [-src gamedata/floppy] [-dst gamedata] [-force]
package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xcifer/johnny-castaway-hd/internal/install"
)

// Files the original installer produces, from the floppy's INSTALL.INS.
// md5 is empty where no reference hash is known.
var files = []struct {
	src, dst, md5 string
}{
	{"SCRANTIC.SC$", "SCRANTIC.SCR", "48b6e36cac54544834c431f6f49c7ada"},
	{"RESOURCE.00$", "RESOURCE.001", "8bb6c99e9129806b5089a39d24228a36"},
	{"RESOURCE.MAP", "RESOURCE.MAP", "374e6d05c5e0acd88fb5af748948c899"},
}

func main() {
	src := flag.String("src", filepath.Join("gamedata", "floppy"), "folder with the floppy's files")
	dst := flag.String("dst", "gamedata", "folder to write the game data to")
	force := flag.Bool("force", false, "write files even if their MD5 doesn't match a known version")
	flag.Parse()

	if err := run(*src, *dst, *force); err != nil {
		fmt.Fprintln(os.Stderr, "jcunpack:", err)
		os.Exit(1)
	}
}

func run(src, dst string, force bool) error {
	type result struct {
		dst  string
		data []byte
	}
	var results []result
	mismatch := false

	// Decode and check everything before writing anything.
	for _, f := range files {
		raw, err := os.ReadFile(filepath.Join(src, f.src))
		if err != nil {
			return err
		}
		data := raw
		if install.IsPacked(raw) {
			p, err := install.Unpack(raw)
			if err != nil {
				return err
			}
			if p.Trailing != 0 {
				fmt.Printf("  note: %s has %d bytes after the compressed stream\n", f.src, p.Trailing)
			}
			data = p.Data
		}

		sum := md5.Sum(data)
		got := hex.EncodeToString(sum[:])
		status := "OK, matches known version"
		if got != f.md5 {
			status = "UNKNOWN VERSION (expected " + f.md5 + ")"
			mismatch = true
		}
		fmt.Printf("%-13s -> %-13s %8d bytes  md5 %s  %s\n", f.src, f.dst, len(data), got, status)
		results = append(results, result{f.dst, data})
	}

	if mismatch && !force {
		return fmt.Errorf("nothing written: some files don't match a known version (use -force to write them anyway)")
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, r := range results {
		path := filepath.Join(dst, r.dst)
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, r.data, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			return err
		}
	}
	fmt.Printf("Wrote %d files to %s\n", len(results), dst)
	return nil
}
