package main

import (
	"errors"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/liudanking/goutil/logutil"
	"github.com/xiehengjian/struct2thrift/idlgen"
	"github.com/xiehengjian/struct2thrift/program"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "struct2thrift",
		Usage:     "generate thrift from go structs",
		UsageText: "struct2thrift -f <file> -s <struct> -o <output>",
		Flags:     flags,
		Action:    run,
		Authors: []*cli.Author{{
			Name:  "xiehengjian",
			Email: "xiehengjian@gmail.com",
		}},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "file, f",
		Usage:    "source file",
		Aliases:  []string{"f"},
		Required: true,
	},
	&cli.StringFlag{
		Name:     "struct, s",
		Usage:    "struct name",
		Aliases:  []string{"s"},
		Required: true,
	},
	&cli.StringFlag{
		Name:     "out, o",
		Usage:    "output file",
		Aliases:  []string{"o"},
		Required: true,
	},
}

func run(c *cli.Context) error {
	file := c.String("file")
	if file == "" {
		file, _ = os.Getwd()
	}
	fi, err := os.Stat(file)
	if err != nil {
		log.Warning("get file info [%s] failed:%v", file, err)
		return err
	}

	pattern := c.String("struct")
	if pattern == "" {
		return errors.New("struct is empty")
	}

	out := c.String("out")
	if out == "" {
		return errors.New("output file is empty")
	}

	if fi.IsDir() {
		return errors.New("can not support dir")
	}

	fset := token.NewFileSet()
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Warning("read [file:%s] failed:%v", file, err)
		return err
	}
	f, err := parser.ParseFile(fset, file, string(data), parser.ParseComments)
	if err != nil {
		log.Warning("parse [file:%s] failed:%v", file, err)
		return err
	}

	typeSpec, err := program.GetStructByName(f, pattern)

	idls, err := idlgen.Generate(f, typeSpec)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(out, []byte(strings.Join(idls, "\n\n")), 0666)
}
