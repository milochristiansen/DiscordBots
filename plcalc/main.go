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

// PLCalc: Production Line Calculator for Spire Race.
package main

import "strings"
import "time"
import "fmt"

import "github.com/bwmarrin/discordgo"

import "github.com/milochristiansen/axis2"
import "github.com/milochristiansen/axis2/sources"

// https://discordapp.com/oauth2/authorize?client_id=402586923765465109&scope=bot&permissions=2048
var (
	APIKey string

	WrethChan   = "340499300184489986"
	KasgyreChan = "340499239962542080"
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

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	for {
		time.Sleep(1 * time.Hour)
	}
	//dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	switch {
	case strings.HasPrefix(m.Content, "!reload"):
		fs := new(axis2.FileSystem)
		fs.Mount("data", sources.NewOSDir("./data"), true)
		loadConfig(fs)
		s.ChannelMessageSend(m.ChannelID, "(Re)loaded data files.")
	case strings.HasPrefix(m.Content, "!debug"):
		side := getSide(m.ChannelID)
		side.Debug = !side.Debug
		s.ChannelMessageSend(m.ChannelID, "Debug mode toggled.")
	case strings.HasPrefix(m.Content, "!help"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!help"))
		if line == "" {
			s.ChannelMessageSend(m.ChannelID, HelpShort)
			return
		}

		if line == "ids" {
			spires := []string{}
			for id := range getSide(m.ChannelID).Spires {
				spires = append(spires, id)
			}
			parts := []string{}
			for id := range getSide(m.ChannelID).Parts {
				parts = append(parts, id)
			}

			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(HelpIDs,
				strings.Join(spires, ", "), getSide(m.ChannelID).Name, strings.Join(parts, ", ")))
			return
		}

		s.ChannelMessageSend(m.ChannelID, HelpLong)
		return
	case strings.HasPrefix(m.Content, "!spires"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!spires"))
		if len(line) > 0 && (line[0] == '+' || line[0] == '-') {
			state := line[0] == '+'
			for _, spire := range strings.Split(line[1:], ",") {
				spire = strings.TrimSpace(spire)
				_, ok := getSide(m.ChannelID).Spires[spire]
				if !ok {
					s.ChannelMessageSend(m.ChannelID, "Invalid spire ID: "+spire+" Ignoring.")
					continue
				}
				getSide(m.ChannelID).SpireList[spire] = state
			}

			spires := []string{}
			for spire, ok := range getSide(m.ChannelID).SpireList {
				if !ok {
					continue
				}

				if spire == "Tweak" {
					spire = spire + "(" + getSide(m.ChannelID).Spires["Tweak"].Prod.String() + ")"
				}
				spires = append(spires, spire)
			}
			s.ChannelMessageSend(m.ChannelID, "Spires set: "+strings.Join(spires, ", "))
			return
		}

		if strings.TrimSpace(line) == "" {
			spires := []string{}
			for spire, ok := range getSide(m.ChannelID).SpireList {
				if !ok {
					continue
				}

				if spire == "Tweak" {
					spire = spire + "(" + getSide(m.ChannelID).Spires["Tweak"].Prod.String() + ")"
				}
				spires = append(spires, spire)
			}
			s.ChannelMessageSend(m.ChannelID, strings.Join(spires, ", "))
			return
		}

		getSide(m.ChannelID).SpireList = map[string]bool{}
		for _, spire := range strings.Split(line, ",") {
			spire = strings.TrimSpace(spire)
			_, ok := getSide(m.ChannelID).Spires[spire]
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Invalid spire ID: "+spire+" Ignoring.")
				return
			}
			getSide(m.ChannelID).SpireList[spire] = true
		}

		spires := []string{}
		for spire, ok := range getSide(m.ChannelID).SpireList {
			if !ok {
				continue
			}

			if spire == "Tweak" {
				spire = spire + "(" + getSide(m.ChannelID).Spires["Tweak"].Prod.String() + ")"
			}
			spires = append(spires, spire)
		}
		s.ChannelMessageSend(m.ChannelID, "Spires set: "+strings.Join(spires, ", "))
		return
	case strings.HasPrefix(m.Content, "!pattern"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!pattern"))

		patterns := strings.Split(line, ",")
		cost := &price{}
		bonuses := map[string]*price{}
		for _, pattern := range patterns {
			parts := parsePattern(pattern)
			if parts == nil {
				s.ChannelMessageSend(m.ChannelID, "Error parsing pattern.")
				return
			}

			partDump := "-\n"
			for _, id := range parts {
				part, ok := getSide(m.ChannelID).Parts[id]
				if !ok {
					s.ChannelMessageSend(m.ChannelID, "Invalid part ID.")
					return
				}
				ok, dump := part.calc(cost, bonuses, m.ChannelID, "> ")
				if !ok {
					s.ChannelMessageSend(m.ChannelID, "Invalid part ID.")
					return
				}
				partDump += dump
			}

			if getSide(m.ChannelID).Debug {
				s.ChannelMessageSend(m.ChannelID, partDump)
			}
		}

		out := "-\nRaw Cost:\n\t`" + cost.String() + "`"

		for id, bonus := range bonuses {
			bonusDef, ok := getSide(m.ChannelID).Bonuses[id]
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Invalid bonus ID.")
				return
			}

			runBonus(cost, bonus, bonusDef.Script)
			out += "\nAfter Bonus:" + bonusDef.Name + "\n\t`" + cost.String() + "`"
		}

		ok, result := calcCOWS(cost, m.ChannelID)
		if !ok {
			s.ChannelMessageSend(m.ChannelID, "Invalid spire list.")
			return
		}

		out += "\nFinal COWS Score:\n\t`" + result.String() + "`"
		s.ChannelMessageSend(m.ChannelID, out)
		return
	case strings.HasPrefix(m.Content, "!tweak"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!tweak"))
		ok, prod := parseCOWS(line)
		if !ok {
			s.ChannelMessageSend(m.ChannelID, "Invalid COWS specifier.")
			return
		}
		getSide(m.ChannelID).Spires["Tweak"].Prod = prod
		s.ChannelMessageSend(m.ChannelID, "Tweak set: "+prod.String())
		return
	case strings.HasPrefix(m.Content, "!calc"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!calc"))
		result, ok := runExpr(line)
		if !ok {
			s.ChannelMessageSend(m.ChannelID, "Invalid expression.")
			return
		}
		s.ChannelMessageSend(m.ChannelID, result)
		return
	default:
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!"))

		ok, cows := parseCOWS(line)
		if ok {
			ok, result := calcCOWS(cows, m.ChannelID)
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Invalid spire list.")
				return
			}

			s.ChannelMessageSend(m.ChannelID, "`"+result.String()+"`")
			return
		}
		return
	}
}
