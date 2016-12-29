package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/jmcvetta/neoism"
	"github.com/juju/errors"
	"github.com/kurrik/twittergo"
)

func (p *Engine) tweetsGet(apiURL string, maxID uint64, params map[string]string) (twittergo.Timeline, error) {
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	var results = twittergo.Timeline{}

	for {
		if maxID != 0 {
			query.Set("max_id", fmt.Sprintf("%v", maxID))
		}

		endpoint := fmt.Sprintf(apiURL, query.Encode())
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, errors.Annotate(err, "create request")
		}

		resp, err := p.client.SendRequest(req)
		if err != nil {
			return nil, errors.Annotate(err, "send request")
		}

		if err = resp.Parse(&results); err != nil {
			if p.handleRateLimitError(err) {
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

func (p *Engine) tweetsImport(db *neoism.Database, tweets twittergo.Timeline, props neoism.Props) error {
	logger.Infof("import %d new tweets", len(tweets))

	props["tweets"] = tweets
	cq := neoism.CypherQuery{
		Statement:  CYPHER_TWEETS_IMPORT,
		Parameters: props,
	}

	if err := db.Cypher(&cq); err != nil {
		return errors.Annotate(err, "db query")
	}

	return nil
}

func (p *Engine) AddTweets() error {
	logger.Info("add tweets")
	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	screenName, err := p.cnf.ScreenName()
	if err != nil {
		return err
	}

	params := map[string]string{
		"count":       "200",
		"screen_name": screenName,
	}

	props := neoism.Props{
		"mention_type": "normal",
	}

	maxID, err := p.getMaxID(db, CYPHER_TWEETS_MAX_ID, screenName)
	if err != nil {
		return errors.Annotate(err, "tweets get max id")
	}

	tweets, err := p.tweetsGet(TW_PATH_USER_TIMELINE, maxID, params)
	if err != nil {
		return errors.Annotate(err, "get tweets")
	}

	for len(tweets) > 0 {
		maxID = tweets[len(tweets)-1].Id() - 1
		if err = p.tweetsImport(db, tweets, props); err != nil {
			return errors.Annotate(err, "import tweets")
		}

		tweets, err = p.tweetsGet(TW_PATH_USER_TIMELINE, maxID, params)
		if err != nil {
			return errors.Annotate(err, "get tweets")
		}
	}

	return nil
}
