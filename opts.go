package main

type CommandLineOpts struct {
	InputFiles []string `short:"i" long:"input" description:"Input GBS files"`
	OutDir     string   `short:"o" long:"outdir" description:"Output directory" default:"./"`
	Version    bool     `short:"v" long:"version" description:"Show version and exit"`
}
