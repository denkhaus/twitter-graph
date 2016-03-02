package main

import "github.com/codegangsta/cli"

func exec(ctx *cli.Context, fn func(eng *Engine) error) {
	cnf, err := NewConfig(ctx)
	if err != nil {
		logger.Fatalf("config error:%s", err)
	}

	eng := NewEngine(cnf)
	if err := fn(eng); err != nil {
		logger.Fatalf("exec error:%s", err)
	}
}
