package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/jmcvetta/neoism"
	"github.com/juju/errors"
)

func (p *Engine) usersGet(apiURL string, cursor int64, params map[string]string) ([]interface{}, int64, error) {
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	query.Set("cursor", fmt.Sprintf("%v", cursor))
	results := make(map[string]interface{})

	for {
		endpoint := fmt.Sprintf(apiURL, query.Encode())
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, 0, errors.Annotate(err, "create request")
		}

		resp, err := p.client.SendRequest(req)
		if err != nil {
			return nil, 0, errors.Annotate(err, "send request")
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

	users := results["users"].([]interface{})
	cursor = int64(results["next_cursor"].(float64))
	return users, cursor, nil
}

func (p *Engine) usersImport(db *neoism.Database, users []interface{}, props neoism.Props) error {
	logger.Infof("import %d users", len(users))

	props["users"] = users
	cq := neoism.CypherQuery{
		Statement:  CYPHER_FOLLOWERS_IMPORT,
		Parameters: props,
	}

	if err := db.Cypher(&cq); err != nil {
		return errors.Annotate(err, "db query")
	}

	return nil
}

func (p *Engine) AddFollowers() error {
	logger.Info("add followers")
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
		"screen_name": screenName,
	}

	users, cursor, err := p.usersGet(TW_PATH_FOLLOWERS_TIMELINE, -1, params)
	if err != nil {
		return errors.Annotate(err, "get followers")
	}

	for len(users) > 0 {
		if err = p.usersImport(db, users, props); err != nil {
			return errors.Annotate(err, "import followers")
		}

		users, cursor, err = p.usersGet(TW_PATH_FOLLOWERS_TIMELINE, cursor, params)
		if err != nil {
			return errors.Annotate(err, "get followers")
		}
	}

	return nil
}
