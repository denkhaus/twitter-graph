package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/juju/errors"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	_ "gopkg.in/cq.v1"
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

func (p *Engine) createClient() {
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

func (p *Engine) AddUser() error {
	p.createClient()

	return nil
}

func (p *Engine) getTweets(screenName string, maxID uint64) (twittergo.Timeline, error) {

	query := url.Values{}
	query.Set("count", "100")
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
	}

	return results, nil
}

func (p *Engine) tweetsImport(db *sql.DB, tweets twittergo.Timeline) error {	
	stmt, err := db.Prepare(CYPHER_TWEETS_IMPORT)
	if err != nil {
		return  errors.Annotate(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query(tweets)
	if err != nil {
		return  errors.Annotate(err, "db query")
	}
	defer rows.Close()
	return nil
}
	
func (p *Engine) tweetsGetMaxID(screenName string) (uint64, error) {
	db, err := sql.Open("neo4j-cypher", p.cnf.neoHost)
	if err != nil {
		return 0, errors.Annotate(err, "open database")
	}
	defer db.Close()

	stmt, err := db.Prepare(CYPHER_TWEETS_MAX_ID)
	if err != nil {
		return 0, errors.Annotate(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query(screenName)
	if err != nil {
		return 0, errors.Annotate(err, "db query")
	}
	defer rows.Close()

	var maxID uint64
	err = rows.Scan(&maxID)
	if err != nil {
		return 0, errors.Annotate(err, "scan query")
	}

	return maxID, nil
}

func (p *Engine) AddTweets() error {
	p.createClient()
	
	db, err := sql.Open("neo4j-cypher", p.cnf.neoHost)
	if err != nil {
		return errors.Annotate(err, "open database")
	}
	defer db.Close()

	screenName, err := p.cnf.ScreenName()
	if err != nil {
		return err
	}

	maxID, err := p.tweetsGetMaxID(screenName)
	if err != nil {
		return errors.Annotate(err, "tweets get max id")
	}

	tweets, err := p.getTweets(screenName, maxID)
	if err != nil {
		return errors.Annotate(err, "get tweets")
	}

	for len(tweets) >0{
		if err = p.tweetsImport(db, tweets); err != nil {
			return errors.Annotate(err, "import tweets")
		}

		for _, tw := range tweets {
			if tw.Id() > maxID {
				maxID = tw.Id()
			}
		}

		tweets, err = p.getTweets(screenName, maxID)
		if err != nil {
			return errors.Annotate(err, "get tweets")
		}
	}

	return nil
}
