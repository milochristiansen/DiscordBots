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

create table if not exists Messages (
	MID text,
	CID text,
	AID text,
	Typ int
);
`

var Queries = map[string]*queryHolder{
	"ChannelInsert": &queryHolder{`insert into Channels (ID) values (?);`, nil},
	"ChannelRemove": &queryHolder{`delete from Channels where ID = ?;`, nil},
	"ChannelList":   &queryHolder{`select ID from Channels;`, nil},

	"FilterInsert": &queryHolder{`insert into UserFilters (User, Filter) values (?, ?);`, nil},
	"FilterRemove": &queryHolder{`delete from UserFilters where (User = ? and Filter = ?);`, nil},
	"FilterList":   &queryHolder{`select User, Filter from UserFilters where (?1 = "" or User = ?1);`, nil},

	"MessageInsert": &queryHolder{`insert into Messages (CID, MID, AID, Typ) values (?, ?, ?, ?);`, nil},
	"MessageRemove": &queryHolder{`delete from Messages where AID = ? and Typ = ?;`, nil},
	"MessageList":   &queryHolder{`select CID, MID, AID, Typ from Messages;`, nil},
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

func addMessage(cid, mid, aid string, alert bool) error {
	t := 0
	if alert {
		t = 1
	}
	_, err := Queries["MessageInsert"].Preped.Exec(cid, mid, aid, t)
	return err
}

func removeMessage(aid string, alert bool) error {
	t := 0
	if alert {
		t = 1
	}
	_, err := Queries["MessageRemove"].Preped.Exec(aid, t)
	return err
}

func getMessages() ([2]map[string][]event, error) {
	rows, err := Queries["MessageList"].Preped.Query()
	if err != nil {
		return [2]map[string][]event{}, err
	}
	defer rows.Close()

	events := [2]map[string][]event{map[string][]event{}, map[string][]event{}}
	for rows.Next() {
		typ, aid := 0, ""
		f := event{}
		err := rows.Scan(&f.CID, &f.MID, &aid, &typ)
		if err != nil {
			return [2]map[string][]event{}, err
		}

		// Make sure bad DB data doesn't crash us.
		if typ != 0 {
			typ = 1
		}
		events[typ][aid] = append(events[typ][aid], f)
	}
	return events, nil
}

type event struct {
	CID string
	MID string
}

func init() {
	var err error
	DB, err = sql.Open("sqlite3", "file:wfalert.db")
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
