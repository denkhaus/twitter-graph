package main

import (
	"github.com/denkhaus/neoism"
	"github.com/juju/errors"
)

func (p *Engine) AddMentions() error {
	logger.Info("add mention tweets")
	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	screenName, err := p.cnf.ScreenName()
	if err != nil {
		return err
	}

	params := map[string]string{
		"count":               "200",
		"screen_name":         screenName,
		"exclude_replies":     "false",
		"contributor_details": "true",
	}

	props := neoism.Props{
		"mention_type": "mention_search",
	}

	maxID, err := p.getMaxID(db, CYPHER_MENTIONS_MAX_ID, screenName)
	if err != nil {
		return errors.Annotate(err, "mentions get max id")
	}

	mentions, err := p.tweetsGet(TW_PATH_MENTIONS_TIMELINE, maxID, params)
	if err != nil {
		return errors.Annotate(err, "get tweets")
	}

	for len(mentions) > 0 {
		maxID = mentions[len(mentions)-1].Id() - 1
		if err = p.tweetsImport(db, mentions, props); err != nil {
			return errors.Annotate(err, "import mention tweets")
		}

		mentions, err = p.tweetsGet(TW_PATH_MENTIONS_TIMELINE, maxID, params)
		if err != nil {
			return errors.Annotate(err, "get tweets")
		}

	}

	return nil
}
