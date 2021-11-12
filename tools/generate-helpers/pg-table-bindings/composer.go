package main

import (
	"fmt"
	"strconv"
	"strings"
)

func commaJoin(s []string) string {
	return strings.Join(s, ",")
}

func countToPlaceholder(count int) string {
	placeholders := make([]string, 0, count)
	for i := 0; i < count; i++ {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
	}
	return commaJoin(placeholders)
}

type InsertComposer struct {
	Table string
	SQL []string
	Excluded []string
	Getters  []string
}

func (ic *InsertComposer) AddSQL(s string) {
	ic.SQL = append(ic.SQL, s)
}

func (ic *InsertComposer) AddExcluded(s string) {
	ic.Excluded = append(ic.Excluded, s)
}

func (ic *InsertComposer) AddGetters(s string) {
	ic.Getters = append(ic.Getters, s)
}

func (ic *InsertComposer) Query() string {
	sql := commaJoin(ic.SQL)
	excluded := commaJoin(ic.Excluded)
	getters := commaJoin(ic.Getters)
	placeholder := countToPlaceholder(len(ic.Getters))

	return fmt.Sprintf("insert into %s(%s) values(%s) on conflict update %s: %+v", ic.Table, sql, placeholder, excluded, getters)
}

func (ic *InsertComposer) Combine(ic2 *InsertComposer) {
	ic.SQL = append(ic.SQL, ic2.SQL...)
	ic.Excluded = append(ic.Excluded, ic2.Excluded...)
	ic.Getters = append(ic.Getters, ic2.Getters...)
}