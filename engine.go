package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/denkhaus/neoism"
	"github.com/juju/errors"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

const (
	TW_PATH_VERIFY_CREDENTIALS = "/1.1/account/verify_credentials.json"
	TW_PATH_USER_TIMELINE      = "/1.1/statuses/user_timeline.json"
)

type Engine struct {
	cnf    *Config
	client *twittergo.Client
}

func NewEngine(cnf *Config) *Engine {
	eng := &Engine{
		cnf: cnf,
	}

	return eng
}

func (p *Engine) ensureTwitterClient() {
	if p.client != nil {
		return
	}

	config := &oauth1a.ClientConfig{
		ConsumerKey:    p.cnf.twitterConsumerKey,
		ConsumerSecret: p.cnf.twitterConsumerSecret,
	}
	user := oauth1a.NewAuthorizedConfig(
		p.cnf.twitterUserKey,
		p.cnf.twitterUserSecret,
	)
	p.client = twittergo.NewClient(config, user)
}

func (p *Engine) openNeoDB() (*neoism.Database, error) {
	logger.Info("open db connection")
	if p.cnf.neoUsername != "" && p.cnf.neoPassword == "" {
		return nil, errors.New("invalid neo4j credentials")
	}

	hostPort := fmt.Sprintf("%s:%d", p.cnf.neoHost, p.cnf.neoPort)
	db, err := neoism.Connect(hostPort, p.cnf.neoUsername, p.cnf.neoPassword)
	if err != nil {
		return nil, errors.Annotate(err, "open database")
	}

	return db, nil
}

func (p *Engine) AddUser() error {
	p.ensureTwitterClient()

	return nil
}

func (p *Engine) tweetsGet(screenName string, maxID uint64) (twittergo.Timeline, error) {
	query := url.Values{}
	query.Set("count", "200")
	query.Set("screen_name", screenName)

	var results = twittergo.Timeline{}
	minwait := time.Duration(10) * time.Second

	for {
		if maxID != 0 {
			query.Set("max_id", fmt.Sprintf("%v", maxID))
		}

		endpoint := fmt.Sprintf("/1.1/statuses/user_timeline.json?%v", query.Encode())
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, errors.Annotate(err, "create request")
		}

		resp, err := p.client.SendRequest(req)
		if err != nil {
			return nil, errors.Annotate(err, "send request")
		}

		if err = resp.Parse(&results); err != nil {
			if rle, ok := err.(twittergo.RateLimitError); ok {
				dur := rle.Reset.Sub(time.Now()) + time.Second
				if dur < minwait {
					dur = minwait
				}

				logger.Infof("Rate limited. Reset at %v. Waiting for %v\n", rle.Reset, dur)
				time.Sleep(dur)
				continue
			} else {
				logger.Errorf("Problem parsing response: %v\n", err)
			}
		}

		if resp.HasRateLimit() {
			logger.Infof("ratelimit: %v calls available", resp.RateLimitRemaining())
		}
		break
	}

	return results, nil
}

func (p *Engine) tweetsImport(db *neoism.Database, tweets twittergo.Timeline) error {
	logger.Infof("import %d new tweets", len(tweets))
	cq := neoism.CypherQuery{
		Statement:  CYPHER_TWEETS_IMPORT,
		Parameters: neoism.Props{"tweets": tweets},
	}

	if err := db.Cypher(&cq); err != nil {
		return errors.Annotate(err, "db query")
	}

	return nil
}

func (p *Engine) tweetsGetMaxID(screenName string) (uint64, error) {
	db, err := p.openNeoDB()
	if err != nil {
		return 0, errors.Annotate(err, "open database")
	}

	var maxIDData []interface{}
	cq := neoism.CypherQuery{
		Statement:  CYPHER_TWEETS_MAX_ID,
		Parameters: neoism.Props{"screen_name": screenName},
		Result:     &maxIDData,
	}

	if err := db.Cypher(&cq); err != nil {
		return 0, errors.Annotate(err, "db query")
	}

	mp := maxIDData[0].(map[string]interface{})
	if mp["max_id"] != nil {
		return uint64(mp["max_id"].(float64)), nil
	}

	return 0, nil
}

func (p *Engine) AddTweets() error {
	logger.Info("add tweets")
	p.ensureTwitterClient()

	db, err := p.openNeoDB()
	if err != nil {
		return errors.Annotate(err, "open database")
	}

	screenName, err := p.cnf.ScreenName()
	if err != nil {
		return err
	}

	maxID, err := p.tweetsGetMaxID(screenName)
	if err != nil {
		return errors.Annotate(err, "tweets get max id")
	}

	tweets, err := p.tweetsGet(screenName, maxID)
	if err != nil {
		return errors.Annotate(err, "get tweets")
	}

	for len(tweets) > 0 {
		maxID = tweets[len(tweets)-1].Id() - 1
		if err = p.tweetsImport(db, tweets); err != nil {
			return errors.Annotate(err, "import tweets")
		}

		tweets, err = p.tweetsGet(screenName, maxID)
		if err != nil {
			return errors.Annotate(err, "get tweets")
		}
	}

	return nil
}
