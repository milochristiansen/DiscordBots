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
import "time"
import "fmt"

import "github.com/mmcdole/gofeed"

import "github.com/bwmarrin/discordgo"

// https://discordapp.com/oauth2/authorize?client_id=402521174384574464&scope=bot&permissions=133120
var (
	APIKey string
	Site   = "https://ceruleanscrawling.wordpress.com"
	Feeds  = []Feed{
		{"/category/summus-proelium/feed", []string{"543593314746761228"}},
		{"/feed", []string{"383419886250098691"}},
	}
)

type Feed struct {
	URL      string
	Channels []string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Spin up the server.

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + APIKey)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(onConnect)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	fp := gofeed.NewParser()
	for {
		stories, err := getStories()
		if err != nil {
			fmt.Println("DB Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		for _, fdata := range Feeds {
			feed, err := fp.ParseURL(Site + fdata.URL)
			if err != nil {
				fmt.Println("Error reading RSS feed:", err)
				break
			}

			for _, item := range feed.Items {
				if !stories[item.Link] {
					fmt.Println("New Post: " + item.Link)
					if item.PublishedParsed != nil {
						addStory(item.Title, item.Link, item.PublishedParsed.Unix())
					} else {
						addStory(item.Title, item.Link, 0)
					}
					stories[item.Link] = true // Needed so that if the next feed in the list has this too it will suppress it.

					for _, id := range fdata.Channels {
						_, err := dg.ChannelMessageSend(id, "@everyone New Post: "+item.Link)
						if err != nil {
							fmt.Println("Error sending message to:", id, err)
						}
					}
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
	//dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// perm, err := s.State.UserChannelPermissions(m.Author.ID, m.ChannelID)
	// if err != nil {
	// 	return
	// }
	// isAdmin := perm&discordgo.PermissionAdministrator != 0 || perm&discordgo.PermissionManageServer != 0 || perm&discordgo.PermissionManageChannels != 0

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
		s.ChannelMessageSend(m.ChannelID, "Try: `Hey Herbie!`. Herbie may also do fun things if you wish him a happy birthday at the right time of year...")
	default:
		t, msg := time.Now(), strings.ToLower(m.Content)
		// September 4th, the day Flick throws Herbie through the portal.
		if t.Month() == time.September && t.Day() == 4 {
			if strings.Contains(msg, "happy") && strings.Contains(msg, "birthday") && strings.Contains(msg, "herbie") {
				s.ChannelMessageSend(m.ChannelID, "Herbie seems pleased with your greeting.")
			}
		}
	}
}

func onConnect(s *discordgo.Session, r *discordgo.Ready) {
	// Discard the error, it doesn't hurt anything if this fails.
	_ = s.UpdateStatus(-1, "Type Herbie? for help.")
}
