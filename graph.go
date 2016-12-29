package main

import (
	"net/url"
	//	"github.com/davecgh/go-spew/spew"
	"strconv"

	"github.com/jmcvetta/neoism"
	"github.com/juju/errors"
)

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

func (p *Engine) MaintainGraph() error {
	logger.Info("maintain graph")

	db, err := p.ensureClients()
	if err != nil {
		return errors.Annotate(err, "ensure clients")
	}

	res, err := p.execQuery(db, CYPHER_NEED_GRAPH_UPDATE, nil)
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

			logger.Infof("update user #%s following relations", idStr)

			props := neoism.Props{
				"id": idStr,
			}

			logger.Infof("remove following relations for user #%s", idStr)
			_, err = p.execQuery(db, CYPHER_REMOVE_FOLLOWS_REL, props)
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

func toString(arr []int64) []string {

	ret := make([]string, len(arr))
	for idx, val := range arr {
		ret[idx] = strconv.FormatInt(val, 10)
	}

	return ret
}
