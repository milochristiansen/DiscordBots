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
	WarframeEndpoint = "https://api.warframestat.us/pc/"
	//WarframeEndpoint = "https://api.warframestat.us/pc/invasions"
)

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

		messages, err := getMessages()
		if err != nil {
			fmt.Println("DB Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Alerts
		r, err := http.Get(WarframeEndpoint + "alerts")
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

		// Invasions
		r, err = http.Get(WarframeEndpoint + "invasions")
		if err != nil {
			if r != nil {
				r.Body.Close()
			}
			fmt.Println("HTTP GET Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		dec = json.NewDecoder(r.Body)
		invasions := []*InvasionData{}
		err = dec.Decode(&invasions)
		r.Body.Close()
		if err != nil {
			fmt.Println("Payload Decode Error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Send/update alert/invasion messages.
		for _, alert := range alerts {
			message, ok := messages[1][alert.ID]
			if !ok {
				sendMessage(dg, channels, filters, alert)
				continue
			}
			// update messages
			editMessages(dg, message, alert, false)
			delete(messages[1], alert.ID)
		}
		for _, invasion := range invasions {
			message, ok := messages[0][invasion.ID]
			if !ok {
				sendMessage(dg, channels, filters, invasion)
				continue
			}
			// update messages
			editMessages(dg, message, invasion, false)
			delete(messages[0], invasion.ID)
		}

		// Kill orphans
		for typ := 0; typ < 2; typ++ {
			alert := typ == 1
			for aid, messages := range messages[typ] {
				editMessages(dg, messages, nil, false)
				removeMessage(aid, alert)
			}
		}

		time.Sleep(1 * time.Minute)
	}
	//dg.Close()
}

func sendMessage(s *discordgo.Session, channels []string, filters []userFilter, item Embedable) {
	msg := item.AsEmbed(false)
	for _, id := range channels {
		mdat, err := s.ChannelMessageSendEmbed(id, msg)
		if err != nil {
			fmt.Println("Error sending message to:", id, err)
			continue
		}
		aid, alert := item.GetID()
		err = addMessage(id, mdat.ID, aid, alert)
		if err != nil {
			fmt.Println("DB Error:", err)
			continue
		}
	}
	for _, filter := range filters {
		if !strings.Contains(strings.ToLower(item.FilterString()), filter.Filter) {
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
		aid, alert := item.GetID()
		err = addMessage(ch.ID, mdat.ID, aid, alert)
		if err != nil {
			fmt.Println("DB Error:", err)
			continue
		}
	}
}

func editMessages(s *discordgo.Session, messages []event, item Embedable, log bool) {
	if log {
		//fmt.Printf("%#v\n", item)
	}

	for _, message := range messages {
		if item == nil {
			m, err := s.ChannelMessage(message.CID, message.MID)
			if err != nil {
				fmt.Println("Error reading old message:", err)
				continue
			}

			if len(m.Embeds) == 0 {
				fmt.Println("Message with no embed:", message.MID, err)
				continue
			}

			embed := m.Embeds[0]
			embed.Color = 0xff0000
			embed.Fields = nil

			_, err = s.ChannelMessageEditEmbed(message.CID, message.MID, embed)
			if err != nil {
				fmt.Println("Error editing message to:", message.MID, err)
			}
			continue
		}

		_, err := s.ChannelMessageEditEmbed(message.CID, message.MID, item.AsEmbed(log))
		if err != nil {
			fmt.Println("Error editing message to:", message.MID, err)
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
