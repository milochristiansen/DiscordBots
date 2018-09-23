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

// Simple bot for the Vintage Story Discord.
package main

import "strings"
import "time"
import "fmt"

import "github.com/bwmarrin/discordgo"

// https://discordapp.com/oauth2/authorize?client_id=485596564455424003&scope=bot&permissions=2048
var (
	APIKey string

	FromChannel = "418404936275984405"
	ToChannel   = "484635648712638475"
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
		time.Sleep(1 * time.Hour)
	}
	//dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != FromChannel {
		return
	}

	if strings.Contains(m.Content, "www.youtube.com/watch") ||
		strings.Contains(m.Content, "youtu.be/") ||
		strings.Contains(m.Content, "twitch.tv/") {

		s.ChannelMessageSend(ToChannel, "<@"+m.Author.ID+">: "+m.Content)
		return
	}
}

func onConnect(s *discordgo.Session, r *discordgo.Ready) {
	// Discard the error, it doesn't hurt anything if this fails.
	_ = s.UpdateStatus(-1, "Vintage Story")
}
