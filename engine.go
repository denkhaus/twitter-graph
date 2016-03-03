package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
"strings"
	"github.com/juju/errors"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"github.com/jmcvetta/neoism"
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

func (p *Engine) openDatabase() (*neoism.Database, error){
	
	var userInfo string		
			if p.cnf.neoUsername != ""{
	userInfo = fmt.Sprintf("%s:%s", p.cnf.neoUsername, p.cnf.neoPassword)
	
	if len(strings.Split(userInfo, ":")) != 2{
		return nil, errors.New("invalid neo4j credentials")
	}  		
	
	}
	
	
	db, err := neoism.Connect()
	if err != nil {
		return errors.Annotate(err, "open database")
	}
	defer db.Close()


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

func (p *Engine) tweetsImport(db *neoism.Database, tweets twittergo.Timeline) error {	
	stmt, err := db.Prepare(CYPHER_TWEETS_IMPORT)
	if err != nil {
		return  errors.Annotate(err, "db prepare")
	}
	defer stmt.Close()

	_, err := stmt.Exec(tweets)
	if err != nil {
		return  errors.Annotate(err, "db exec")
	}
	
	return nil
}
	
func (p *Engine) tweetsGetMaxID(screenName string) (uint64, error) {

	db, err := p.openDatabase()
	if err != nil {
		return 0, errors.Annotate(err, "open database")
	}

	
	
	
var maxID uint64
cq := neoism.CypherQuery{    
    Statement: CYPHER_TWEETS_MAX_ID,
    Parameters: neoism.Props{"screen_name": screenName},    
	Result: &maxID,
}
	
	if err := db.Cypher(&cq); err != nil {
		return 0, errors.Annotate(err, "db query")
	}
	
	return maxID, nil
}

func (p *Engine) AddTweets() error {
	p.createClient()
	
	db, err := neoism.Connect(p.cnf.neoHost)
	if err != nil {
		return 0, errors.Annotate(err, "open database")
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
