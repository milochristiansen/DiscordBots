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

//
package main

import "encoding/json"
import "net/http"
import "strings"
import "time"
import "fmt"

import "github.com/bwmarrin/discordgo"

// https://discordapp.com/oauth2/authorize?client_id=472522276101816320&scope=bot&permissions=2048
var (
	APIKey           string
	WarframeEndpoint = "https://api.warframestat.us/pc/alerts"
)

/*
TODO:

* Alerts for invasions
* Look into using embeds.

*/

func main() {
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

	for {
		//fmt.Println("Checking alerts...")
		channels, err := getChannels()
		if err != nil {
			fmt.Println("DB Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}
		filters, err := getFilters("")
		if err != nil {
			fmt.Println("DB Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		activeAlerts, err := getActiveAlerts()
		if err != nil {
			fmt.Println("DB Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		r, err := http.Get(WarframeEndpoint)
		if err != nil {
			if r != nil {
				r.Body.Close()
			}
			fmt.Println("HTTP GET Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		dec := json.NewDecoder(r.Body)
		alerts := []*AlertData{}
		err = dec.Decode(&alerts)
		r.Body.Close()
		if err != nil {
			fmt.Println("Payload Decode Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		handled := map[string]bool{}
		for _, item := range alerts {
			handled[item.ID] = true

			// Alert has expired and is in DB.
			if activeAlerts[item.ID] && item.Expired {
				err = removeActiveAlert(item.ID)
				if err != nil {
					fmt.Println("DB Error:", err)
				}
				fmt.Println("expired:", item.ID)
				editMessage(dg, item.ID, item, true)
				continue
			}

			// Alert has expired and is not in DB
			if item.Expired {
				fmt.Println("expired, no DB:", item.ID)
				editMessage(dg, item.ID, item, true) // Just in case. Probably won't do anything.
				continue
			}

			// Alert already sent, update the message.
			if _, ok := activeAlerts[item.ID]; ok {
				//fmt.Println("already sent:", item.Mission.Reward.Desc)
				editMessage(dg, item.ID, item, false)
				continue
			}

			err = addActiveAlert(item.ID)
			if err != nil {
				fmt.Println("DB Error:", err)
				continue
			}

			fmt.Println("sending:", item.ID)
			sendMessage(dg, channels, filters, item)
		}

		// Check for orphan items in the DB.
		for id := range activeAlerts {
			if handled[id] {
				continue
			}
			fmt.Println("orphan:", id)
			err = removeActiveAlert(id)
			if err != nil {
				fmt.Println("DB Error:", err)
			}
			editMessage(dg, id, nil, true)
		}

		time.Sleep(1 * time.Minute)
	}
	//dg.Close()
}

func sendMessage(s *discordgo.Session, channels []string, filters []userFilter, item *AlertData) {
	msg := item.AsEmbed(false)
	for _, id := range channels {
		mdat, err := s.ChannelMessageSendEmbed(id, msg)
		if err != nil {
			fmt.Println("Error sending message to:", id, err)
			continue
		}
		err = addActiveAlertMessage(item.ID, id, mdat.ID)
		if err != nil {
			fmt.Println("DB Error:", err)
			continue
		}
	}
	for _, filter := range filters {
		if !strings.Contains(strings.ToLower(item.Mission.Reward.Desc), filter.Filter) {
			continue
		}

		ch, err := s.UserChannelCreate(filter.UID)
		if err != nil {
			fmt.Println("Error sending message to:", ch.ID, err)
			continue
		}
		mdat, err := s.ChannelMessageSendEmbed(ch.ID, msg)
		if err != nil {
			fmt.Println("Error sending message to:", ch.ID, err)
			continue
		}
		err = addActiveAlertMessage(item.ID, ch.ID, mdat.ID)
		if err != nil {
			fmt.Println("DB Error:", err)
			continue
		}
	}
}

func editMessage(s *discordgo.Session, aid string, item *AlertData, log bool) {
	if log {
		//fmt.Printf("%#v\n", item)
	}

	messages, err := getActiveAlertMessages(aid)
	if err != nil {
		fmt.Println("DB Error:", err)
		return
	}

	for _, message := range messages {
		if item == nil {
			m, err := s.ChannelMessage(message[0], message[1])
			if err != nil {
				fmt.Println("Error reading old message:", err)
				continue
			}

			if len(m.Embeds) == 0 {
				fmt.Println("Message with no embed:", message[0], err)
				continue
			}

			embed := m.Embeds[0]
			embed.Color = 0xff0000
			embed.Fields = nil

			_, err = s.ChannelMessageEditEmbed(message[0], message[1], embed)
			if err != nil {
				fmt.Println("Error editing message to:", message[0], err)
			}
			continue
		}

		_, err := s.ChannelMessageEditEmbed(message[0], message[1], item.AsEmbed(log))
		if err != nil {
			fmt.Println("Error editing message to:", message[0], err)
			continue
		}
	}
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

	command := parseCommand(m.Content)
	if len(command) < 1 {
		return
	}

	switch command[0] {
	case "&help":
		s.ChannelMessageSend(m.ChannelID, "Try: `&filter \"filter text\"`, `&rmfilter \"filter text\"`, `&filters`, `&post` (admin only), or `&nopost` (admin only)")
	case "&post":
		if !isAdmin {
			s.ChannelMessageSend(m.ChannelID, "Sorry, you are not the server admin.")
			return
		}

		err := addChannel(m.ChannelID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error, check server logs.")
			fmt.Println("Channel add error:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Now posting alerts here.")
	case "&nopost":
		if !isAdmin {
			s.ChannelMessageSend(m.ChannelID, "Sorry, you are not the server admin.")
			return
		}

		err := removeChannel(m.ChannelID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error, check server logs.")
			fmt.Println("Channel remove error:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "No longer posting alerts here.")
	case "&filter":
		if len(command) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Argument needed.")
			return
		}
		err := addFilter(m.Author.ID, command[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error, check server logs.")
			fmt.Println("Filter add error:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Added filter: "+command[1])
	case "&rmfilter":
		if len(command) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Argument needed.")
			return
		}
		err := removeFilter(m.Author.ID, command[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error, check server logs.")
			fmt.Println("Filter remove error:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Removed filter: "+command[1])
	case "&filters":
		filters, err := getFilters(m.Author.ID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error, check server logs.")
			fmt.Println("Filter list error:", err)
			return
		}
		msg := "Your Filters:"
		for _, filter := range filters {
			msg += fmt.Sprint("\n", filter.Filter)
		}
		s.ChannelMessageSend(m.ChannelID, msg)
	}
}

func onConnect(s *discordgo.Session, r *discordgo.Ready) {
	// Discard the error, it doesn't hurt anything if this fails.
	_ = s.UpdateStatus(-1, "Warframe | &help")
}

// For when strings.Split just isn't good enough...
func parseCommand(in string) []string {
	out := make([]string, 0)
	var buf []byte

	skipwhite := true
	quotes := false
	for i := range in {
		b := in[i]

		// Quoted things
		if quotes && b != '"' {
			buf = append(buf, b)
			continue
		}
		if b == '"' {
			quotes = !quotes
			out = append(out, string(buf))
			buf = buf[0:0]
			continue
		}

		// White space
		if skipwhite && (b == ' ' || b == '\t') {
			continue
		}
		if b == ' ' || b == '\t' {
			skipwhite = true
			continue
		}
		if skipwhite {
			skipwhite = false
			if len(buf) > 0 {
				out = append(out, string(buf))
			}
			buf = nil
		}

		// Everything else
		buf = append(buf, b)
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}
