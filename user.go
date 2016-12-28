package main

import (
	//	"github.com/davecgh/go-spew/spew"
	"time"

	"github.com/denkhaus/neoism"
	"github.com/juju/errors"
)

func (p *Engine) completeUsers(db *neoism.Database) error {

	res, err := p.execQuery(db, CYPHER_USERS_NEED_COMPLETION, nil)
	if err != nil {
		return errors.Annotate(err, "exec users need completion")
	}

	ids := res.FilterResultsBy("id").ToInt64Slice()

	if len(ids) > 0 {
		logger.Infof("%d user ids need completion -> fetch", len(ids))
		twUsers, err := p.api.GetUsersLookupByIds(ids, nil)
		if err != nil {
			return errors.Annotate(err, "lookup users by ids")
		}

		props := neoism.Props{
			"users": twUsers,
		}

		logger.Infof("got data for %d users -> apply", len(twUsers))
		if _, err := p.execQuery(db, CYPHER_USERS_UPDATE_BY_ID, props); err != nil {
			return errors.Annotate(err, "users update by id")
		}
	}

	return nil
}

func (p *Engine) CompleteUsers() error {
	logger.Info("complete users")

	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	for {
		if err := p.completeUsers(db); err != nil {
			return errors.Annotate(err, "complete users")
		}

		time.Sleep(10 * time.Second)
	}

}