package main

import (
	"fmt"
	"gbs2gb/constants"
	"gbs2gb/gbs"
	"gbs2gb/utils"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path"
	"strings"
)

func main() {
	opts := CommandLineOpts{}
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if w, ok := err.(*flags.Error); ok {
			if w.Type == flags.ErrHelp {
				return
			}
			log.Fatalln(err)
		}
	}

	if opts.Version {
		fmt.Printf("gbs2gb %s\n", constants.GitVersion)
		return
	}

	if err := os.MkdirAll(opts.OutDir, 0750); err != nil {
		log.Fatalf("Can't create output directory: %v", err)
	}

	_, inputBase := path.Split(opts.InputFile)
	inputNoext := strings.TrimSuffix(inputBase, path.Ext(inputBase))
	outputFile := path.Join(opts.OutDir, fmt.Sprintf("%s.gb", inputNoext))

	log.Printf("→ %s", inputBase)

	gbsBytes, err := utils.ReadAllBytes(opts.InputFile)
	if err != nil {
		log.Println(err)
		return
	}

	err = gbs.MakeGB(gbsBytes, outputFile)
	if err != nil {
		log.Printf("Couldn't create GB file:")
		log.Println(err)
	}

	log.Printf("Successfully created %s", outputFile)
}
