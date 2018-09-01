/*
Copyright 2018 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package main

import "strings"

import _ "github.com/mattn/go-sqlite3"
import "database/sql"

var DB *sql.DB

var InitCode = `
create table if not exists Channels (
	ID text
);

create table if not exists UserFilters (
	User text,
	Filter text
);

create table if not exists ActiveAlerts (
	ID text
);

create table if not exists AlertMessages (
	MID text,
	CID text,
	AID text
);
`

var Queries = map[string]*queryHolder{
	"ChannelInsert": &queryHolder{`insert into Channels (ID) values (?);`, nil},
	"ChannelRemove": &queryHolder{`delete from Channels where ID = ?;`, nil},
	"ChannelList":   &queryHolder{`select ID from Channels;`, nil},

	"FilterInsert": &queryHolder{`insert into UserFilters (User, Filter) values (?, ?);`, nil},
	"FilterRemove": &queryHolder{`delete from UserFilters where (User = ? and Filter = ?);`, nil},
	"FilterList":   &queryHolder{`select User, Filter from UserFilters where (?1 = "" or User = ?1);`, nil},

	"AlertInsert": &queryHolder{`insert into ActiveAlerts (ID) values (?);`, nil},
	"AlertRemove": &queryHolder{`delete from ActiveAlerts where ID = ?;`, nil},
	"AlertList":   &queryHolder{`select ID from ActiveAlerts;`, nil},

	"MessageInsert": &queryHolder{`insert into AlertMessages (AID, CID, MID) values (?, ?, ?);`, nil},
	"MessageRemove": &queryHolder{`delete from AlertMessages where AID = ?;`, nil},
	"MessageList":   &queryHolder{`select CID, MID from AlertMessages where AID = ?;`, nil},
}

func addChannel(id string) error {
	_, err := Queries["ChannelInsert"].Preped.Exec(id)
	return err
}

func removeChannel(id string) error {
	_, err := Queries["ChannelRemove"].Preped.Exec(id)
	return err
}

func getChannels() ([]string, error) {
	rows, err := Queries["ChannelList"].Preped.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []string{}
	for rows.Next() {
		f := new(string)
		err := rows.Scan(f)
		if err != nil {
			return nil, err
		}
		channels = append(channels, *f)
	}
	return channels, nil
}

func addFilter(uid, filter string) error {
	_, err := Queries["FilterInsert"].Preped.Exec(uid, strings.ToLower(filter))
	return err
}

func removeFilter(uid, filter string) error {
	_, err := Queries["FilterRemove"].Preped.Exec(uid, strings.ToLower(filter))
	return err
}

// Pass "" as the UID to list all.
func getFilters(uid string) ([]userFilter, error) {
	rows, err := Queries["FilterList"].Preped.Query(uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filters := []userFilter{}
	for rows.Next() {
		f := userFilter{}
		err := rows.Scan(&f.UID, &f.Filter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

type userFilter struct {
	UID    string
	Filter string
}

func addActiveAlert(id string) error {
	_, err := Queries["AlertInsert"].Preped.Exec(id)
	return err
}

func removeActiveAlert(id string) error {
	_, err := Queries["AlertRemove"].Preped.Exec(id)
	if err != nil {
		return err
	}
	_, err = Queries["MessageRemove"].Preped.Exec(id)
	return err
}

func addActiveAlertMessage(aid, cid, mid string) error {
	_, err := Queries["MessageInsert"].Preped.Exec(aid, cid, mid)
	return err
}

func getActiveAlerts() (map[string]bool, error) {
	rows, err := Queries["AlertList"].Preped.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := map[string]bool{}
	for rows.Next() {
		f := new(string)
		err := rows.Scan(f)
		if err != nil {
			return nil, err
		}
		alerts[*f] = true
	}
	return alerts, nil
}

func getActiveAlertMessages(id string) ([][2]string, error) {
	rows, err := Queries["MessageList"].Preped.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := [][2]string{}
	for rows.Next() {
		c, m := new(string), new(string)
		err := rows.Scan(c, m)
		if err != nil {
			return nil, err
		}
		messages = append(messages, [2]string{*c, *m})
	}
	return messages, nil
}

func init() {
	var err error
	DB, err = sql.Open("sqlite3", "file:feeds.db")
	if err != nil {
		panic(err)
	}

	_, err = DB.Exec(InitCode)
	if err != nil {
		panic(err)
	}

	for _, v := range Queries {
		err := v.Init()
		if err != nil {
			panic(err)
		}
	}
}

type queryHolder struct {
	Code   string
	Preped *sql.Stmt
}

func (q *queryHolder) Init() error {
	var err error
	q.Preped, err = DB.Prepare(q.Code)
	return err
}
