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
import "strconv"
import "time"
import "fmt"

import "github.com/bwmarrin/discordgo"

import "github.com/milochristiansen/axis2"
import "github.com/milochristiansen/axis2/sources"

// https://discordapp.com/oauth2/authorize?client_id=402586923765465109&scope=bot&permissions=2048
var (
	APIKey = "NDAyNTg2OTIzNzY1NDY1MTA5.DT650g.lo6hhry2MMQ1Xq4_-LaOw-mRh_0"

	WrethChan   = "340499300184489986"
	KasgyreChan = ""
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
	case strings.HasPrefix(m.Content, "!help"):
		spires := []string{}
		for id := range getSide(m.ChannelID).Spires {
			spires = append(spires, id)
		}
		patterns := []string{}
		for id := range getSide(m.ChannelID).Cores {
			patterns = append(patterns, id)
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(`
Set Spire: `+"`"+`!spires <list of comma separated spire IDs>`+"`"+`
-------------------------------------------------------------------------------

Set the spires used to calculate COWS. The special symbol '@' is the
"spire" used by the Tweak Production ('!tweak') command, and 'Home' is all
the home spires.

In addition to specifying the entire list of spires you want, you can
activate or deactivate spires with the following syntax:

`+"```"+`
!spires + <list of comma separated spire IDs>
!spires - <list of comma separated spire IDs>
`+"```"+`

If no spires are specified this command will print the current list.

Default spire list:

`+"```"+`
@, Home
`+"```"+`

Valid spire IDs:

`+"```"+`
%v
`+"```"+`

Calculate Design: `+"`"+`!pattern <design ID>`+"`"+`
-------------------------------------------------------------------------------

Calculate the COWS for a named pattern. This pattern needs to be
defined in the data files.

If you wish, you can specify a count for a pattern, or specify multiple
patterns to calculate together. For example, calculate the cost of one
pattern "Test-A" and two pattern "Test-B":

`+"```"+`
# Test-A, Test-B:2
`+"```"+`

The following patterns are currently loaded for %v:

`+"```"+`
%v
`+"```"+`

Tweak Production: `+"`"+`!tweak 0c,0o,0w,0s`+"`"+`
-------------------------------------------------------------------------------

Set a modifier for spire production. For this to take effect the '@'
spire must be in the spire list.

Note that the COWS numbers may be partly specified, specified in any
order, or even left blank. Any missing value is set to 0.
	   
Calculate from raw COWS: `+"`"+`! 0c,0o,0w,0s`+"`"+`
-------------------------------------------------------------------------------

Calculate production line COWS from a raw COWS value.

Note that the COWS numbers may be partly specified, specified in any
order, or even left blank. Any missing value is set to 0.
`, strings.Join(spires, ", "), getSide(m.ChannelID).Name, strings.Join(patterns, ", ")))
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
			return
		}

		if strings.TrimSpace(line) == "" {
			spires := []string{}
			for spire, ok := range getSide(m.ChannelID).SpireList {
				if !ok {
					continue
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
		return
	case strings.HasPrefix(m.Content, "!pattern"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!pattern"))

		patterns := strings.Split(line, ",")
		cost := &price{}
		bonuses := map[string]*price{}
		for _, pattern := range patterns {
			parts := strings.SplitN(pattern, ":", 2)
			count := 1.0
			var err error
			if len(parts) == 2 {
				count, err = strconv.ParseFloat(parts[1], 64)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Invalid count.")
					return
				}
			}
			mult := &price{C: count, O: count, W: count, S: count}

			core, ok := getSide(m.ChannelID).Cores[strings.TrimSpace(parts[0])]
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Invalid pattern.")
				return
			}

			cost.add(core.Cost.copy().mul(mult))
			for id, bonus := range core.Bonus {
				_, ok := bonuses[id]
				if !ok {
					bonuses[id] = &price{}
				}
				bonuses[id].add(bonus.copy().mul(mult))
			}
			for _, id := range core.Parts {
				part, ok := getSide(m.ChannelID).Parts[id]
				if !ok {
					s.ChannelMessageSend(m.ChannelID, "Invalid part ID.")
					return
				}
				cost.add(part.Cost.copy().mul(mult))
				for id, bonus := range part.Bonus {
					_, ok := bonuses[id]
					if !ok {
						bonuses[id] = &price{}
					}
					bonuses[id].add(bonus.copy().mul(mult))
				}
			}
		}

		out := "Raw Cost:\n\t" + cost.String()

		for id, bonus := range bonuses {
			bonusDef, ok := getSide(m.ChannelID).Bonuses[id]
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Invalid bonus ID.")
				return
			}

			runBonus(cost, bonus, bonusDef.Script)
			out += "\nAfter Bonus:" + bonusDef.Name + "\n\t" + cost.String()
		}

		ok, result := calcCOWS(cost, m.ChannelID)
		if !ok {
			s.ChannelMessageSend(m.ChannelID, "Invalid spire list.")
			return
		}

		out += "\nFinal COWS Score:\n\t" + result.String()
		s.ChannelMessageSend(m.ChannelID, out)
		return
	case strings.HasPrefix(m.Content, "!tweak"):
		line := strings.TrimSpace(strings.TrimPrefix(m.Content, "!tweak"))
		ok, prod := parseCOWS(line)
		if !ok {
			fmt.Println("Invalid COWS specifier.")
		}
		getSide(m.ChannelID).Spires["@"].Prod = prod
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

			s.ChannelMessageSend(m.ChannelID, result.String())
			return
		}

		s.ChannelMessageSend(m.ChannelID, "Invalid input.")
		return
	}
}
