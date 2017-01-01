package main

import (
	"net/url"
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
	//	"github.com/davecgh/go-spew/spew"
	"github.com/jmcvetta/neoism"
	"github.com/juju/errors"
)

func (p *Engine) maintainFollowing(db *neoism.Database) error {
	res, err := p.execQuery(db, CYPHER_NEED_GRAPH_UPDATE_FOLLOWING, nil)
	if err != nil {
		return errors.Annotate(err, "exec following mismatch")
	}

	ids := res.FilterResultsBy("id").ToStringSlice()

	if len(ids) > 0 {

		commonIdCount := 0
		params := url.Values{}

		for _, idStr := range ids {

			if commonIdCount != 0 {
				logger.Infof("%d relations updated", commonIdCount)
			}

			props := neoism.Props{
				"id": idStr,
			}

			logger.Infof("remove following relations for user #%s", idStr)
			_, err = p.execQuery(db, CYPHER_REMOVE_FOLLOWING_REL, props)
			if err != nil {
				return errors.Annotate(err, "exec remove follows relationship")
			}

			commonIdCount = 0
			params.Set("user_id", idStr)
			cursor := "-1"

			for {

				logger.Infof("retrive users user #%s is following", idStr)

				params.Set("cursor", cursor)
				cur, err := p.api.GetFriendsIds(params)
				if err != nil {
					if apiErr, ok := err.(*anaconda.ApiError); ok {

						if ok, tm := apiErr.RateLimitCheck(); ok {
							dur := tm.Sub(time.Now())
							logger.Warnf("rate limit error: wait for %s", dur)
							time.Sleep(dur)
							break
						} else if apiErr.StatusCode > 400 {
							logger.Warnf("received error code %d", apiErr.StatusCode)
							break
						}
					}

					return errors.Annotate(err, "get friends ids")
				}

				idCount := len(cur.Ids)
				if idCount == 0 {
					break
				}

				commonIdCount += idCount
				props = neoism.Props{
					"user_id": idStr,
					"ids":     toString(cur.Ids),
				}

				logger.Infof("merge %d users user #%s is following", idCount, idStr)
				_, err = p.execQuery(db, CYPHER_MERGE_FOLLOWING_IDS, props)
				if err != nil {
					return errors.Annotate(err, "insert following ids")
				}

				cursor = cur.Next_cursor_str
			}
		}
	}

	return nil

}

func (p *Engine) MaintainFollowing() error {
	logger.Info("maintain following")

	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	if err := p.maintainFollowing(db); err != nil {
		return errors.Annotate(err, "maintain following")
	}

	return nil
}

func (p *Engine) maintainFollowers(db *neoism.Database) error {
	res, err := p.execQuery(db, CYPHER_NEED_GRAPH_UPDATE_FOLLOWERS, nil)
	if err != nil {
		return errors.Annotate(err, "exec followers mismatch")
	}

	ids := res.FilterResultsBy("id").ToStringSlice()

	if len(ids) > 0 {

		commonIdCount := 0
		params := url.Values{}

		for _, idStr := range ids {
			if commonIdCount != 0 {
				logger.Infof("%d relations updated", commonIdCount)
			}

			props := neoism.Props{
				"id": idStr,
			}

			logger.Infof("remove followers relations for user #%s", idStr)
			_, err = p.execQuery(db, CYPHER_REMOVE_FOLLOWERS_REL, props)
			if err != nil {
				return errors.Annotate(err, "remove followers relation")
			}

			commonIdCount = 0
			params.Set("user_id", idStr)
			cursor := "-1"

			for {

				logger.Infof("retrive followers of user #%s", idStr)

				params.Set("cursor", cursor)
				cur, err := p.api.GetFollowersIds(params)
				if err != nil {
					if apiErr, ok := err.(*anaconda.ApiError); ok {

						if ok, tm := apiErr.RateLimitCheck(); ok {
							dur := tm.Sub(time.Now())
							logger.Warnf("rate limit error: wait for %s", dur)
							time.Sleep(dur)
							break
						} else if apiErr.StatusCode > 400 {
							logger.Warnf("received error code %d", apiErr.StatusCode)
							break
						}
					}

					return errors.Annotate(err, "get followers ids")
				}

				idCount := len(cur.Ids)
				if idCount == 0 {
					break
				}

				commonIdCount += idCount
				props = neoism.Props{
					"user_id": idStr,
					"ids":     toString(cur.Ids),
				}

				logger.Infof("merge %d followers of user #%s", idCount, idStr)
				_, err = p.execQuery(db, CYPHER_MERGE_FOLLOWERS_IDS, props)
				if err != nil {
					return errors.Annotate(err, "insert followers ids")
				}

				cursor = cur.Next_cursor_str
			}
		}
	}

	return nil

}

func (p *Engine) MaintainFollowers() error {
	logger.Info("maintain followers")

	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	if err := p.maintainFollowers(db); err != nil {
		return errors.Annotate(err, "maintain followers")
	}

	return nil
}

func toString(arr []int64) []string {

	ret := make([]string, len(arr))
	for idx, val := range arr {
		ret[idx] = strconv.FormatInt(val, 10)
	}

	return ret
}
