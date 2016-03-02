package main

import (
	"errors"

	"github.com/codegangsta/cli"
)

type Config struct {
	ctx                   *cli.Context
	neoHost               string
	twitterConsumerKey    string
	twitterConsumerSecret string
	twitterUserKey        string
	twitterUserSecret     string
	screenName            string
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	cnf := &Config{
		ctx: ctx,
	}

	cnf.neoHost = ctx.GlobalString("host")
	if cnf.neoHost == "" {
		return nil, errors.New("neo host is not defined")
	}

	cnf.twitterConsumerKey = ctx.GlobalString("consumer-key")
	if cnf.twitterConsumerKey == "" {
		return nil, errors.New("twitter consumer key is not defined")
	}

	cnf.twitterConsumerSecret = ctx.GlobalString("consumer-secret")
	if cnf.twitterConsumerSecret == "" {
		return nil, errors.New("twitter consumer secret is not defined")
	}

	cnf.twitterUserKey = ctx.GlobalString("user-key")
	if cnf.twitterUserKey == "" {
		return nil, errors.New("twitter user key is not defined")
	}

	cnf.twitterUserSecret = ctx.GlobalString("user-secret")
	if cnf.twitterUserSecret == "" {
		return nil, errors.New("twitter user secret is not defined")
	}

	return cnf, nil
}

func (p *Config) ScreenName() (string, error) {
	p.screenName = p.ctx.GlobalString("screenname")
	if p.screenName == "" {
		return "", errors.New("twitter user secret is not defined")
	}

	return p.screenName, nil
}
