package main

import (
	"github.com/urfave/cli"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type Input struct {
	Filename string
	Format   string
}

type Output struct {
	Filename string
	Format   string
}

type GenOutput struct {
	Format           string
	Directory        string
	Package          string
	PrefixesToRemove []string
}

func create(c *cli.Context) error {
	args := c.Args()
	argsCount := c.NArg()
	var filename string
	if argsCount == 0 {
		filename = "db.ojson"
	} else {
		filename = args.Get(0)
	}

	cmd := &CreateCmd{}
	return cmd.Create(&Output{Filename: filename})
}

func convert(c *cli.Context) error {
	args := c.Args()
	argsCount := c.NArg()
	if argsCount == 0 {
		return cli.NewExitError("source is not set", 1)
	}
	if argsCount == 1 {
		return cli.NewExitError("target is not set", 1)
	}

	input := &Input{
		Filename: args.Get(0),
		Format:   c.String("sourceFormat"),
	}
	output := &Output{
		Filename: args.Get(1),
		Format:   c.String("targetFormat"),
	}

	cmd := &ConvertCmd{}
	return cmd.Convert(input, output)
}

func generate(c *cli.Context) error {
	args := c.Args()
	argsCount := c.NArg()
	if argsCount == 0 {
		return cli.NewExitError("source is not set", 1)
	}
	if argsCount == 1 {
		return cli.NewExitError("target is not set", 1)
	}

	input := &Input{
		Filename: args.Get(0),
		Format:   c.String("sourceFormat"),
	}

	prefixesToRemove := strings.Split(c.String("removePrefix"), ",")

	output := &GenOutput{
		Directory:        args.Get(1),
		Format:           c.String("targetFormat"),
		Package:          c.String("package"),
		PrefixesToRemove: prefixesToRemove,
	}

	cmd := &GenerateCmd{}
	return cmd.Generate(input, output)
}

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = "oct"
	cliApp.Version = "0.0.1"
	cliApp.Compiled = time.Now()
	cliApp.Authors = []cli.Author{
		{Name: "Lechuck Roh"},
	}
	cliApp.Copyright = "(c) 2019 Lechuck Roh"
	cliApp.Usage = "octopus-db-tools"
	cliApp.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "create `filename`",
			Action:  create,
		},
		{
			Name:    "convert",
			Aliases: []string{"c"},
			Usage:   "convert `source` `target`",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "sourceFormat, sf",
					Usage:  "set source format",
					EnvVar: "OCTOPUS_SOURCE_FORMAT",
				},
				cli.StringFlag{
					Name:   "targetFormat, tf",
					Usage:  "set target format",
					EnvVar: "OCTOPUS_TARGET_FORMAT",
				},
			},
			Action: convert,
		},
		{
			Name:    "generate",
			Aliases: []string{"g"},
			Usage:   "generate `source` `target`",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "sourceFormat, sf",
					Usage:  "set source format",
					EnvVar: "OCTOPUS_SOURCE_FORMAT",
				},
				cli.StringFlag{
					Name:   "targetFormat, tf",
					Usage:  "set target format",
					EnvVar: "OCTOPUS_TARGET_FORMAT",
				},
				cli.StringFlag{
					Name:   "package, p",
					Usage:  "set target package name",
					EnvVar: "OCTOPUS_PACKAGE",
				},
				cli.StringFlag{
					Name:   "removePrefix, rp",
					Usage:  "set prefixes to remove. set multiple values with comma separated.",
					EnvVar: "OCTOPUS_REMOVE_PREFIX",
				},
			},
			Action: generate,
		},
	}

	sort.Sort(cli.FlagsByName(cliApp.Flags))
	sort.Sort(cli.CommandsByName(cliApp.Commands))

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}