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

import _ "github.com/mattn/go-sqlite3"
import "database/sql"

var DB *sql.DB

var InitCode = `
create table if not exists ReadStories (
	ID integer primary key,

	Name text collate nocase,
	URL text,

	Published integer
);

create table if not exists Channels (
	ID text
)
`

var Queries = map[string]*queryHolder{
	"StoryInsert":   &queryHolder{`insert into ReadStories (Name, URL, Published) values (?, ?, ?);`, nil},
	"StoryList":     &queryHolder{`select URL from ReadStories;`, nil},
	"ChannelInsert": &queryHolder{`insert into Channels (ID) values (?);`, nil},
	"ChannelRemove": &queryHolder{`delete from Channels where ID = ?;`, nil},
	"ChannelList":   &queryHolder{`select ID from Channels;`, nil},
}

func addChannel(id string) error {
	_, err := Queries["ChannelInsert"].Preped.Exec(id)
	return err
}

func removeChannel(id string) error {
	_, err := Queries["ChannelRemove"].Preped.Exec(id)
	return err
}

func addStory(name, url string, published int64) error {
	_, err := Queries["StoryInsert"].Preped.Exec(name, url, published)
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

func getStories() (map[string]bool, error) {
	rows, err := Queries["StoryList"].Preped.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := map[string]bool{}
	for rows.Next() {
		f := new(string)
		err := rows.Scan(f)
		if err != nil {
			return nil, err
		}
		stories[*f] = true
	}
	return stories, nil
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
