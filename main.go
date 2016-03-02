package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
)

func init() {
	logger = logrus.New()
	logger.Level = logrus.DebugLevel
	logger.Out = os.Stdout
}

func main() {

	app := cli.NewApp()
	app.Name = "twitter-graph"
	app.Usage = "A docker log pump to splunk"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host, d",
			Usage:  "Neo4j host",
			EnvVar: "NEO4_HOST",
			Value:  "http://localhost:7474",
		},

		cli.StringFlag{
			Name:   "consumer-key",
			Usage:  "Twitter Consumer Key",
			EnvVar: "TWITTER_CONSUMER_KEY",
		},

		cli.StringFlag{
			Name:   "consumer-secret",
			Usage:  "Twitter Consumer Secret",
			EnvVar: "TWITTER_CONSUMER_SECRET",
		},

		cli.StringFlag{
			Name:   "user-key",
			Usage:  "Twitter User Key",
			EnvVar: "TWITTER_USER_KEY",
		},

		cli.StringFlag{
			Name:   "user-secret",
			Usage:  "Twitter User Secret",
			EnvVar: "TWITTER_USER_SECRET",
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name: "add",
			Subcommands: []cli.Command{
				cli.Command{
					Name: "user",
					Action: func(ctx *cli.Context) {
						exec(ctx, func(eng *Engine) error {
							return eng.AddUser()
						})
					},
				},
				cli.Command{
					Name: "tweets",
					Action: func(ctx *cli.Context) {
						exec(ctx, func(eng *Engine) error {
							return eng.AddTweets()
						})
					},
				},
			},
		},
	}

	app.Run(os.Args)
}

//logsPump := NewLogsPump(storagePath)

//		closer.Bind(func() {
//			logsPump.Shutdown()
//			logger.Info("terminated")
//		})

//		closer.Checked(func() error {
//			logger.Info("startup ---------------------------------------------")
//			logsPump.RegisterAdapter(NewSplunkAdapter, host)
//			return logsPump.Run()
//		}, true)
