package main

import (
	"fmt"
	"time"

	"net/url"

	"github.com/ChimeraCoder/anaconda"
	"github.com/jmcvetta/neoism"
	"github.com/juju/errors"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

const (
	TW_PATH_VERIFY_CREDENTIALS = "/1.1/account/verify_credentials.json?%v"
	TW_PATH_USER_TIMELINE      = "/1.1/statuses/user_timeline.json?%v"
	TW_PATH_MENTIONS_TIMELINE  = "/1.1/statuses/mentions_timeline.json?%v"
	TW_PATH_FOLLOWERS_TIMELINE = "/1.1/followers/list.json?%v"
	TW_PATH_FOLLOWERS_IDS      = "/1.1/followers/ids.json?%v"
	TW_PATH_USER_LOOKUP        = "/1.1/users/lookup.json?"
)

type Engine struct {
	cnf    *Config
	client *twittergo.Client
	api    *anaconda.TwitterApi
}

func NewEngine(cnf *Config) *Engine {
	eng := &Engine{
		cnf: cnf,
	}

	return eng
}

func (p *Engine) Close() {
	if p.api != nil {
		p.api.Close()
		p.api = nil
	}
}

func (p *Engine) ensureTwitterClient() {
	if p.client != nil {
		return
	}

	anaconda.SetConsumerKey(p.cnf.twitterConsumerKey)
	anaconda.SetConsumerSecret(p.cnf.twitterConsumerSecret)

	p.api = anaconda.NewTwitterApi(
		p.cnf.twitterUserKey,
		p.cnf.twitterUserSecret,
	)

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

	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", p.cnf.neoHost, p.cnf.neoPort),
	}

	if p.cnf.neoUsername != "" {
		u.User = url.UserPassword(p.cnf.neoUsername, p.cnf.neoPassword)
	}

	logger.Debug("neo4j connection string:", u.String())

	db, err := neoism.Connect(u.String())
	if err != nil {
		return nil, errors.Annotate(err, "open database")
	}

	return db, nil
}

func (p *Engine) ensureClients() (*neoism.Database, error) {
	p.ensureTwitterClient()

	db, err := p.openNeoDB()
	if err != nil {
		return nil, errors.Annotate(err, "open database")
	}
	err = p.initDatabase(db)
	return db, err
}

func (p *Engine) initDatabase(db *neoism.Database) error {
	logger.Infof("init database")

	cyphers := []string{
		CYPHER_CONSTRAINT_TWEET,
		CYPHER_CONSTRAINT_USERNAME,
		CYPHER_CONSTRAINT_USERID,
		CYPHER_CONSTRAINT_HASHTAG,
		CYPHER_CONSTRAINT_LINK,
		CYPHER_CONSTRAINT_SOURCE,
	}

	for _, cy := range cyphers {
		cq := neoism.CypherQuery{
			Statement: cy,
		}

		if err := db.Cypher(&cq); err != nil {
			return errors.Annotate(err, "db query")
		}
	}

	return nil
}

func (p *Engine) execQuery(db *neoism.Database, statmnt string, props neoism.Props) (*CypherResult, error) {

	res := NewCypherResult()
	cq := neoism.CypherQuery{
		Statement:  statmnt,
		Result:     &res.Raw,
		Parameters: props,
	}

	if err := db.Cypher(&cq); err != nil {
		return nil, errors.Annotate(err, "db query")
	}

	return res, nil
}
func (p *Engine) handleRateLimitError(err error) bool {
	minwait := time.Duration(10) * time.Second
	if rle, ok := err.(twittergo.RateLimitError); ok {
		dur := rle.Reset.Sub(time.Now()) + time.Second
		if dur < minwait {
			dur = minwait
		}

		logger.Infof("Rate limited. Reset at %v. Waiting for %v\n", rle.Reset, dur)
		time.Sleep(dur)
		return true
	}

	return false
}

func (p *Engine) getMaxID(db *neoism.Database, cypher, screenName string) (uint64, error) {

	var maxIDData []interface{}
	cq := neoism.CypherQuery{
		Statement:  cypher,
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
