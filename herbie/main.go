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

// Herbie: Heretical Edge new post Discord notification bot.
package main

import "io/ioutil"
import "math/rand"
import "strings"
import "flag"
import "time"
import "fmt"

import "github.com/mmcdole/gofeed"

import "github.com/bwmarrin/discordgo"

// https://discordapp.com/oauth2/authorize?client_id=402521174384574464&scope=bot&permissions=133120
var (
	APIKey  string
	RSSPath = "https://ceruleanscrawling.wordpress.com/feed/"
)

func main() {
	flag.StringVar(&RSSPath, "rsspath", RSSPath, "URL for the RSS feed.")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	// Spin up the server.

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + APIKey)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	fp := gofeed.NewParser()
	for {
		// Discord channel IDs are saved in a totally brain-dead csv file. This way if the bot crashes nothing is lost.
		idlist, err := ioutil.ReadFile("herbie.ids")
		if err != nil {
			continue
		}
		ids := strings.Split(string(idlist), ",")

		check := time.Now()

		// Same deal with the last check time.
		last, err := ioutil.ReadFile("herbie.last")
		if err != nil {
			fmt.Println("Last time parse error! Reinitializing system.\nError:", err)
			_ = ioutil.WriteFile("herbie.last", []byte(check.Format(time.RFC3339)), 0666)
			continue
		}

		feed, err := fp.ParseURL(RSSPath)
		if err != nil {
			fmt.Println("Error reading RSS feed:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		lastT, err := time.Parse(time.RFC3339, strings.TrimSpace(string(last)))
		if err != nil {
			fmt.Println("Last time parse error! Reinitializing system.\nError:", err)
			_ = ioutil.WriteFile("herbie.last", []byte(check.Format(time.RFC3339)), 0666)
			continue
		}

		for _, item := range feed.Items {
			if item.PublishedParsed.After(lastT) {
				fmt.Println("New Post: " + item.Link)
				for _, id := range ids {
					id = strings.TrimSpace(id)
					if id == "" {
						continue
					}
					_, err := dg.ChannelMessageSend(id, "@everyone New Post: "+item.Link)
					if err != nil {
						fmt.Println("Error sending message to:", id, err)
					}
				}
			}
		}

		_ = ioutil.WriteFile("herbie.last", []byte(check.Format(time.RFC3339)), 0666)
		time.Sleep(1 * time.Minute)
	}
	//dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	perm, err := s.State.UserChannelPermissions(m.Author.ID, m.ChannelID)
	if err != nil {
		return
	}
	isAdmin := perm&discordgo.PermissionAdministrator != 0 || perm&discordgo.PermissionManageServer != 0 || perm&discordgo.PermissionManageChannels != 0

	switch m.Content {
	case "Hey Herbie!":
		linelist, err := ioutil.ReadFile("herbie.quotes")
		if err != nil {
			return
		}
		lines := strings.Split(string(linelist), "\n")
		nlines := []string{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			nlines = append(nlines, line)
		}
		if len(nlines) > 0 {
			s.ChannelMessageSend(m.ChannelID, nlines[rand.Intn(len(nlines))])
		}
	case "Herbie?":
		s.ChannelMessageSend(m.ChannelID, "Try: `Hey Herbie!`, `Herbie, post here.` (admin only), or `Herbie, stop posting here.` (admin only)")
	case "Herbie, post here.":
		if !isAdmin {
			s.ChannelMessageSend(m.ChannelID, "Herbie glares at you.")
			return
		}

		idlist, _ := ioutil.ReadFile("herbie.ids")
		ids := strings.Split(string(idlist), ",")
		ids = append(ids, m.ChannelID)

		nids := []string{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			nids = append(nids, id)
		}

		_ = ioutil.WriteFile("herbie.ids", []byte(strings.Join(nids, ",")), 0666)
		s.ChannelMessageSend(m.ChannelID, "Herbie seems to perk up a bit.")
	case "Herbie, stop posting here.":
		if !isAdmin {
			s.ChannelMessageSend(m.ChannelID, "Herbie glares at you.")
			return
		}

		idlist, _ := ioutil.ReadFile("herbie.ids")
		ids := strings.Split(string(idlist), ",")

		nids := []string{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" || id == m.ChannelID {
				continue
			}
			nids = append(nids, id)
		}

		_ = ioutil.WriteFile("herbie.ids", []byte(strings.Join(nids, ",")), 0666)
		s.ChannelMessageSend(m.ChannelID, "Herbie looks bored.")
	}
}
