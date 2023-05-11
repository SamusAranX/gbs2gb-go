package main

type CommandLineOpts struct {
	InputFile string `short:"i" long:"input" description:"Input GBS file"`
	OutDir    string `short:"o" long:"outdir" description:"Output directory" default:"./"`
	Version   bool   `short:"v" long:"version" description:"Show version and exit"`
}
