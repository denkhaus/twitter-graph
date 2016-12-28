package main

import (
	"strconv"
)

type CypherResult struct {
	Raw interface{}
}

func NewCypherResult() *CypherResult {
	return &CypherResult{}
}

func (p *CypherResult) ToStringSlice() []string {
	res, ok := p.Raw.([]interface{})
	if !ok {
		panic("cypher result: wrong raw value type")
	}

	ret := []string{}
	for _, val := range res {
		s := val.(string)
		if len(s) > 0 {
			ret = append(ret, s)
		}
	}

	return ret
}

func (p *CypherResult) ToInt64Slice() []int64 {
	res, ok := p.Raw.([]interface{})
	if !ok {
		panic("cypher result: wrong raw value type")
	}

	ret := []int64{}
	for _, val := range res {

		if val != nil {
			switch v := val.(type) {
			case string:
				l, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					panic("conversion error")
				}
				ret = append(ret, l)
			case int64:
				ret = append(ret, v)
			case float64:
				ret = append(ret, int64(v))
			default:
				panic("unhandled type")
			}
		}
	}

	return ret
}

func (p *CypherResult) GetAt(idx int) *CypherResult {
	res, ok := p.Raw.([]interface{})
	if !ok {
		panic("cypher result: wrong raw value type")
	}

	if len(res) > idx {
		return &CypherResult{Raw: res[idx]}
	}

	return nil
}

func (p *CypherResult) FilterResultsBy(name string) *CypherResult {
	res, ok := p.Raw.([]interface{})
	if !ok {
		panic("cypher result: wrong raw value type")
	}

	ret := []interface{}{}
	for _, r := range res {
		mp := r.(map[string]interface{})
		if val, ok := mp[name]; ok {
			ret = append(ret, val)
		}
	}

	return &CypherResult{Raw: ret}
}

func (p *CypherResult) GetProperty(name string) interface{} {

	res, ok := p.Raw.(map[string]interface{})
	if !ok {
		panic("cypher result: wrong raw value type")
	}

	val, ok := res[name]
	if !ok {
		return nil
	}

	return val
}
