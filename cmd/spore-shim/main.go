package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	fc "github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "restore":
		restoreCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: spore-shim <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  restore   Restore a microVM from snapshot files")
}

func restoreCmd(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	var (
		fcBin     = fs.String("fc-bin", "firecracker", "firecracker binary")
		jailerBin = fs.String("jailer-bin", "jailer", "jailer binary")
		socket    = fs.String("socket-path", "", "firecracker socket path")
		id        = fs.String("id", "", "vm id")
	)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "snapshot directory required")
		fs.Usage()
		os.Exit(1)
	}

	dir := fs.Arg(0)
	spec := fc.RestoreSpec{
		MemFile:     filepath.Join(dir, "snapshot.mem"),
		VMStateFile: filepath.Join(dir, "snapshot.vmstate"),
		ConfigFile:  filepath.Join(dir, "snapshot.config"),
		JailerBin:   *jailerBin,
		FCBin:       *fcBin,
		SocketPath:  *socket,
		ID:          *id,
	}

	if err := fc.Restore(context.Background(), spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
