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

import "fmt"
import "time"

import "github.com/bwmarrin/discordgo"

type Embedable interface {
	AsEmbed(log bool) *discordgo.MessageEmbed
	FilterString() string
	GetID() (string, bool)
}

type AlertData struct {
	ID         string           `json:"id"`
	Activation time.Time        `json:"activation"`
	Expiry     time.Time        `json:"expiry"`
	Mission    AlertMissionData `json:"mission"`
	Expired    bool             `json:"expired"`
	ETA        string           `json:"eta"`
}

func (a *AlertData) String() string {
	remaining := ""
	if a.Expiry.Sub(time.Now()).Round(time.Minute) < time.Minute || a.Expired {
		remaining = "*expired*"
	} else {
		remaining = fmt.Sprintf("%dm", a.Expiry.Sub(time.Now()).Round(time.Minute)/time.Minute)
	}
	msg := a.Mission.Type + " at " + a.Mission.Node + " for " + a.Mission.Reward.Desc + ".\nRemaining time: " + remaining
	if a.Activation.After(time.Now()) {
		msg = "Alert: In " + fmt.Sprintf("%dm", a.Activation.Sub(time.Now()).Round(time.Minute)/time.Minute) + "\n" + msg
	} else {
		msg = "Alert:\n" + msg
	}
	return msg
}

func (a *AlertData) AsEmbed(log bool) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{}

	color := 0x00ff00 // Assume green (AKA, "currently running").

	// If it hasn't started yet, add a field with the time till start.
	if a.Activation.After(time.Now()) {
		color = 0x0000ff
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Starting in:",
			Value:  fmt.Sprintf("%dm", a.Activation.Sub(time.Now()).Round(time.Minute)/time.Minute),
			Inline: true,
		})
	}

	if log {
		//fmt.Println(a.Expired, a.Expiry.Sub(time.Now()).Round(time.Minute) < time.Minute)
	}

	// If it has expired change the color to red, otherwise add a field with the time until it ends.
	if a.Expired || a.Expiry.Sub(time.Now()).Round(time.Minute) < time.Minute {
		color = 0xff0000
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Remaining Time:",
			Value:  fmt.Sprintf("%dm", a.Expiry.Sub(time.Now()).Round(time.Minute)/time.Minute),
			Inline: true,
		})
	}

	// fields = append(fields, &discordgo.MessageEmbedField{
	// 	Name:   "Debug:",
	// 	Value:  a.ID,
	// 	Inline: false,
	// })

	return &discordgo.MessageEmbed{
		Color:       color, // Blue when not started, Green while running, Red when finished.
		Title:       "Alert:",
		Description: a.Mission.Type + " at " + a.Mission.Node + " for " + a.Mission.Reward.Desc,
		Fields:      fields,
	}
}

func (a *AlertData) FilterString() string {
	return a.Mission.Reward.Desc
}

func (a *AlertData) GetID() (string, bool) {
	return a.ID, true
}

type AlertMissionData struct {
	Node    string          `json:"node"`
	Type    string          `json:"type"`
	Faction string          `json:"faction"`
	Reward  AlertRewardData `json:"reward"`
}

type AlertRewardData struct {
	Desc string `json:"asString"`
}

type InvasionData struct {
	ID             string          `json:"id"`
	Node           string          `json:"node"`
	Defender       string          `json:"defendingFaction"`
	VSInfestation  bool            `json:"vsInfestation"`
	Activation     time.Time       `json:"activation"`
	AttackerReward AlertRewardData `json:"attackerReward"`
	DefenderReward AlertRewardData `json:"defenderReward"`
	Completed      bool            `json:"completed"`
	ETA            string          `json:"eta"`
}

func (a *InvasionData) AsEmbed(log bool) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{}

	color := 0x00ff00 // Assume green (AKA, "currently running").

	// If it hasn't started yet, add a field with the time till start.
	if a.Activation.After(time.Now()) {
		color = 0x0000ff
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Starting in:",
			Value:  fmt.Sprintf("%dm", a.Activation.Sub(time.Now()).Round(time.Minute)/time.Minute),
			Inline: true,
		})
	}

	if log {
		//fmt.Println()
	}

	// If it has expired change the color to red.
	if a.Completed {
		color = 0xff0000
	}

	// fields = append(fields, &discordgo.MessageEmbedField{
	// 	Name:   "Debug:",
	// 	Value:  a.ID,
	// 	Inline: false,
	// })

	msg := ""
	if a.VSInfestation {
		msg = a.DefenderReward.Desc
	} else {
		msg = a.AttackerReward.Desc + " vs " + a.DefenderReward.Desc
	}

	return &discordgo.MessageEmbed{
		Color:       color, // Blue when not started, Green while running, Red when finished.
		Title:       "Invasion:",
		Description: "At " + a.Node + " for " + msg,
		Fields:      fields,
	}
}

func (a *InvasionData) FilterString() string {
	msg := ""
	if a.VSInfestation {
		msg = a.DefenderReward.Desc
	} else {
		msg = a.AttackerReward.Desc + " vs " + a.DefenderReward.Desc
	}
	return msg
}

func (a *InvasionData) GetID() (string, bool) {
	return a.ID, false
}
