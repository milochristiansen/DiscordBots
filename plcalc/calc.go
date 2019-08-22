/*
Copyright 2017-2018 by Milo Christiansen

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

import "github.com/milochristiansen/lua"
import "github.com/milochristiansen/lua/lmodbase"
import "github.com/milochristiansen/lua/lmodstring"
import "github.com/milochristiansen/lua/lmodtable"
import "github.com/milochristiansen/lua/lmodmath"

import "github.com/milochristiansen/axis2"

import "github.com/bwmarrin/discordgo"

import "encoding/json"
import "strconv"
import "strings"
import "fmt"

// Base Production Values
// Identical for both sides. Used to populate the "Home" pseudo-spire.
var BaseProd = &price{
	C: 13,
	O: 17,
	W: 19,
	S: 10,
}

// Base Scaling Factors
var BaseFactors = &price{
	C: 1 / (BaseProd.C / 60),
	O: 1 / (BaseProd.O / 60),
	W: 1 / (BaseProd.W / 60),
	S: 1 / (BaseProd.S / 60),
}

type sideDef struct {
	Name      string
	SpireList map[string]bool
	Spires    map[string]*spire
	Bonuses   map[string]*prodBonus
	Parts     map[string]*prodPart
	Debug     bool
}

const (
	WrethSide   = "340499300184489986"
	KasgyreSide = "340499239962542080"
)

var HomeSpire = &spire{
	Name: "Home Spires",
	Desc: "A fake 'spire' representing the home spires for each side.",
	ID:   "Home",

	Prod: BaseProd,
}

var Sides = map[string]*sideDef{
	WrethSide: &sideDef{
		Name: "Wreth",
		SpireList: map[string]bool{
			"Tweak": true,
			"Home":  true,
		},
		Spires: map[string]*spire{
			"Tweak": {
				Name: "Income Modifier",
				Desc: "A fake 'spire' used to preview income changes.",
				ID:   "Tweak",

				Prod: &price{},
			},
			"Home": HomeSpire,
		},
		Bonuses: map[string]*prodBonus{},
		Parts:   map[string]*prodPart{},
	},
	KasgyreSide: &sideDef{
		Name: "Kasgyre",
		SpireList: map[string]bool{
			"Tweak": true,
			"Home":  true,
		},
		Spires: map[string]*spire{
			"Tweak": {
				Name: "Income Modifier",
				Desc: "A fake 'spire' used to preview income changes.",
				ID:   "Tweak",

				Prod: &price{},
			},
			"Home": HomeSpire,
		},
		Bonuses: map[string]*prodBonus{},
		Parts:   map[string]*prodPart{},
	},
	"DEFAULT": &sideDef{
		Name: "Default side",
		SpireList: map[string]bool{
			"Tweak": true,
			"Home":  true,
		},
		Spires: map[string]*spire{
			"Tweak": {
				Name: "Income Modifier",
				Desc: "A fake 'spire' used to preview income changes.",
				ID:   "Tweak",

				Prod: &price{},
			},
			"Home": HomeSpire,
		},
		Bonuses: map[string]*prodBonus{},
		Parts:   map[string]*prodPart{},
	},
}

func getSide(side string) *sideDef {
	s, ok := Sides[side]
	if !ok {
		return Sides["DEFAULT"]
	}
	return s
}

func checkSpires(spires, side string) bool {
	for _, spire := range strings.Split(spires, ",") {
		_, ok := getSide(side).Spires[strings.TrimSpace(spire)]
		if !ok {
			return false
		}
	}
	return true
}

func calcCOWS(cows *price, side string) (bool, *price) {
	prod := &price{}
	s := getSide(side)
	for spire, ok := range s.SpireList {
		if !ok {
			continue
		}

		s, ok := s.Spires[spire]
		if !ok {
			return false, nil
		}
		prod.add(s.Prod)
	}

	factors := BaseFactors.copy().div(prod.div(BaseProd))
	return true, cows.copy().mul(factors)
}

func loadConfig(fs *axis2.FileSystem) {
	fmt.Println("Loading data files:")

	oSides := Sides
	Sides = map[string]*sideDef{
		WrethSide: &sideDef{
			Name:      "Wreth",
			SpireList: oSides[WrethSide].SpireList,
			Spires: map[string]*spire{
				"Tweak": oSides[WrethSide].Spires["Tweak"],
				"Home":  HomeSpire,
			},
			Bonuses: map[string]*prodBonus{},
			Parts:   map[string]*prodPart{},
		},
		KasgyreSide: &sideDef{
			Name:      "Kasgyre",
			SpireList: oSides[KasgyreSide].SpireList,
			Spires: map[string]*spire{
				"Tweak": oSides[KasgyreSide].Spires["Tweak"],
				"Home":  HomeSpire,
			},
			Bonuses: map[string]*prodBonus{},
			Parts:   map[string]*prodPart{},
		},
		"DEFAULT": &sideDef{
			Name:      "Default side",
			SpireList: oSides["DEFAULT"].SpireList,
			Spires: map[string]*spire{
				"Tweak": oSides["DEFAULT"].Spires["Tweak"],
				"Home":  HomeSpire,
			},
			Bonuses: map[string]*prodBonus{},
			Parts:   map[string]*prodPart{},
		},
	}

	for _, filepath := range fs.ListFiles("data") {
		fmt.Println(filepath)
		loadConfigFile(fs, filepath)
	}
}

func loadToSides(file string, action func(side *sideDef)) {
	switch {
	case strings.HasPrefix(file, "wreth"):
		side := getSide(WrethSide)
		action(side)
	case strings.HasPrefix(file, "kasgyre"):
		side := getSide(KasgyreSide)
		action(side)
	default:
		for _, side := range Sides {
			action(side)
		}
	}
}

func loadConfigFile(fs *axis2.FileSystem, filepath string) {
	rdr, err := fs.Read("data/" + filepath)
	if err != nil {
		panic(err)
	}
	defer rdr.Close()

	switch GetExt(filepath) {
	case ".spire":
		vs := []*spire{}
		err := json.NewDecoder(rdr).Decode(&vs)
		if err != nil {
			panic(err)
		}
		loadToSides(filepath, func(side *sideDef) {
			for _, v := range vs {
				side.Spires[v.ID] = v
			}
		})
	case ".bonus":
		vs := []*prodBonus{}
		err := json.NewDecoder(rdr).Decode(&vs)
		if err != nil {
			panic(err)
		}
		loadToSides(filepath, func(side *sideDef) {
			for _, v := range vs {
				script, err := fs.ReadAll("data/" + v.Script)
				if err != nil {
					panic(err)
				}
				v.Script = string(script)
				side.Bonuses[v.ID] = v
			}
		})
	case ".part":
		vs := []*prodPart{}
		err := json.NewDecoder(rdr).Decode(&vs)
		if err != nil {
			panic(err)
		}
		loadToSides(filepath, func(side *sideDef) {
			for _, v := range vs {
				side.Parts[v.ID] = v
			}
		})
	default:
		// Error
	}
}

type price struct {
	C, O, W, S float64
}

func (p *price) String() string {
	if p == nil {
		return "0.00c + 0.00o + 0.00w + 0.00s = 0.00"
	}
	return fmt.Sprintf("%.2fc + %.2fo + %.2fw + %.2fs = %.2f", Round(p.C, 0.01), Round(p.O, 0.01), Round(p.W, 0.01), Round(p.S, 0.01), Round(p.total(), 0.01))
}

func (p *price) total() float64 {
	return p.C + p.O + p.W + p.S
}

func (p *price) copy() *price {
	if p == nil {
		p = &price{}
	}

	return &price{
		C: p.C,
		O: p.O,
		W: p.W,
		S: p.S,
	}
}

func (p *price) add(b *price) *price {
	if b == nil {
		return p
	}
	p.C += b.C
	p.O += b.O
	p.W += b.W
	p.S += b.S
	return p
}

func (p *price) sub(b *price) *price {
	if b == nil {
		return p
	}
	p.C -= b.C
	p.O -= b.O
	p.W -= b.W
	p.S -= b.S
	return p
}

func (p *price) mul(b *price) *price {
	if b == nil {
		return p
	}
	p.C *= b.C
	p.O *= b.O
	p.W *= b.W
	p.S *= b.S
	return p
}

func (p *price) div(b *price) *price {
	if b == nil {
		return p
	}
	p.C /= b.C
	p.O /= b.O
	p.W /= b.W
	p.S /= b.S
	return p
}

type spire struct {
	Name string
	Desc string
	ID   string

	Prod *price
}

type prodBonus struct {
	Name string
	Desc string
	ID   string

	// Lua script, has two globals:
	//	IN: Total cost of item as a table with C,O,W,S keys.
	//	BONUS: Total value of bonus items from all parts as a table with C,O,W,S keys.
	// Return value should be a table with C,O,W,S keys used as the final value.
	//
	// These scripts are called in undefined order. Each gets the results from the one
	// before, and passes its results to the next in line.
	Script string
}

type prodPart struct {
	Name string
	Desc string
	ID   string

	Cost  *price
	Bonus map[string]*price

	Parts []string
}

func (part *prodPart) calc(cost *price, count int, bonus map[string]*price, side, lvl string, s *discordgo.Session) (bool, string) {
	mult := &price{C: float64(count), O: float64(count), W: float64(count), S: float64(count)}
	cost.add(part.Cost.copy().mul(mult))
	for id, bdat := range part.Bonus {
		_, ok := bonus[id]
		if !ok {
			bonus[id] = &price{}
		}
		bonus[id].add(bdat.copy().mul(mult))
	}

	sidedef := getSide(side)

	partDump := lvl + part.ID + ": " + part.Cost.String() + "\n"
	for _, id := range part.Parts {
		cpart, ok := sidedef.Parts[id]
		if !ok {
			s.ChannelMessageSend(side, "Invalid part ID: `"+id+"`")
			return false, ""
		}
		ok, dump := cpart.calc(cost, count, bonus, side, lvl+"> ", s)
		if !ok {
			return false, ""
		}
		partDump += dump
	}
	return true, partDump
}

func parseCOWS(cows string) (bool, *price) {
	rtn := &price{}

	// The empty string is a valid cows number, just one equal to 0.
	if strings.TrimSpace(cows) == "" {
		return true, rtn
	}

	parts := strings.Split(cows, ",")
	for _, part := range parts {
		part := strings.TrimSpace(part)
		if len(part) < 2 {
			return false, nil
		}

		v, err := strconv.ParseFloat(part[:len(part)-1], 64)
		if err != nil {
			return false, nil
		}

		switch part[len(part)-1] {
		case 'c':
			rtn.C = v
		case 'o':
			rtn.O = v
		case 'w':
			rtn.W = v
		case 's':
			rtn.S = v
		default:
			return false, nil
		}
	}

	return true, rtn
}

func parsePattern(pattern, side string, s *discordgo.Session) (*prodPart, int) {
	ids := strings.Split(pattern, ";")

	partList := map[string]int{}
	firstPart := ""
	firstCount := 0
	for i := 0; i < len(ids); i++ {
		parts := strings.SplitN(ids[i], ":", 2)
		pcount := 1
		var err error
		if len(parts) == 2 {
			pcount, err = strconv.Atoi(parts[1])
			if err != nil {
				s.ChannelMessageSend(side, "Count is not a valid integer: `"+parts[1]+"`")
				return nil, 0
			}
		}

		id := strings.TrimSpace(parts[0])
		if id == "" {
			continue
		}

		_, ok := getSide(side).Parts[id]
		if !ok {
			if i == 0 {
				s.ChannelMessageSend(side, "First part ID does not exist: `"+id+"`")
			} else {
				s.ChannelMessageSend(side, "Part specified for removal does not exist: `"+id+"`")
			}
			return nil, 0
		}
		if i == 0 {
			firstPart = id
			firstCount = pcount
			continue
		}
		partList[id] += pcount
	}

	// We already made sure this exists.
	base := getSide(side).Parts[firstPart]
	ret := &prodPart{
		Cost:  base.Cost,
		Bonus: base.Bonus,
	}

	existingParts := map[string]int{}
	for _, part := range base.Parts {
		existingParts[part]++
	}
	for part := range existingParts {
		existingParts[part] -= partList[part]
	}
	for part := range existingParts {
		for i := 0; i < existingParts[part]; i++ {
			ret.Parts = append(ret.Parts, part)
		}
	}

	return ret, firstCount
}

var l = lua.NewState()

func init() {
	// Load standard modules
	l.Push(lmodbase.Open)
	l.Call(0, 0)
	l.Push(lmodstring.Open)
	l.Call(0, 0)
	l.Push(lmodtable.Open)
	l.Call(0, 0)
	l.Push(lmodmath.Open)
	l.Call(0, 0)
}

func runExpr(expr string) (string, bool) {
	err := l.LoadText(strings.NewReader("return "+expr), "", 0)
	if err != nil {
		return "", false
	}

	err = l.PCall(0, 1)
	if err != nil {
		return "", false
	}

	rtn := l.ToString(-1)
	l.Pop(1)
	return rtn, true
}

func runBonus(cows, bonus *price, script string) {
	l.PushIndex(lua.GlobalsIndex)
	l.Push("IN")
	l.NewTable(0, 4)
	l.Push("C")
	l.Push(cows.C)
	l.SetTableRaw(-3)
	l.Push("O")
	l.Push(cows.O)
	l.SetTableRaw(-3)
	l.Push("W")
	l.Push(cows.W)
	l.SetTableRaw(-3)
	l.Push("S")
	l.Push(cows.S)
	l.SetTableRaw(-3)
	l.SetTableRaw(-3)

	l.Push("BONUS")
	l.NewTable(0, 4)
	l.Push("C")
	l.Push(bonus.C)
	l.SetTableRaw(-3)
	l.Push("O")
	l.Push(bonus.O)
	l.SetTableRaw(-3)
	l.Push("W")
	l.Push(bonus.W)
	l.SetTableRaw(-3)
	l.Push("S")
	l.Push(bonus.S)
	l.SetTableRaw(-3)
	l.SetTableRaw(-3)

	l.Pop(1)

	err := l.LoadText(strings.NewReader(script), "", 0)
	if err != nil {
		panic(err)
	}

	err = l.PCall(0, 1)
	if err != nil {
		panic(err)
	}

	l.Push("C")
	l.GetTable(-2)
	f, ok := l.TryFloat(-1)
	if ok {
		cows.C = f
	}
	l.Pop(1)
	l.Push("O")
	l.GetTable(-2)
	f, ok = l.TryFloat(-1)
	if ok {
		cows.O = f
	}
	l.Pop(1)
	l.Push("W")
	l.GetTable(-2)
	f, ok = l.TryFloat(-1)
	if ok {
		cows.W = f
	}
	l.Pop(1)
	l.Push("S")
	l.GetTable(-2)
	f, ok = l.TryFloat(-1)
	if ok {
		cows.S = f
	}
	l.Pop(2)
}

func GetExt(name string) string {
	// Find the last part of the extension
	i := len(name) - 1
	for i >= 0 {
		if name[i] == '.' {
			return name[i:]
		}
		i--
	}
	return ""
}

func Round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}
