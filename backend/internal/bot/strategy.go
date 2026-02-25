package bot

import (
	"math/rand"
	"sort"

	"github.com/game-playzui/tienlen-server/internal/game"
	"github.com/game-playzui/tienlen-server/internal/models"
)

type Difficulty int

const (
	DiffEasy   Difficulty = 0
	DiffMedium Difficulty = 1
	DiffHard   Difficulty = 2
)

type Play struct {
	Cards     []models.Card
	ComboType models.CombinationType
}

func ChoosePlay(hand []models.Card, table *TableState, diff Difficulty) *Play {
	if table == nil || table.IsEmpty {
		return chooseOpening(hand, diff)
	}
	return chooseBeat(hand, table, diff)
}

type TableState struct {
	IsEmpty   bool
	Cards     []models.Card
	ComboType models.CombinationType
}

// chooseOpening picks a combination to lead with when the table is clear.
func chooseOpening(hand []models.Card, diff Difficulty) *Play {
	combos := decomposeHand(hand)
	if len(combos) == 0 {
		return &Play{Cards: []models.Card{lowestCard(hand)}, ComboType: models.ComboSingle}
	}

	sort.Slice(combos, func(i, j int) bool {
		return comboStrength(combos[i]) < comboStrength(combos[j])
	})

	switch diff {
	case DiffEasy:
		idx := rand.Intn(len(combos))
		return combos[idx]
	case DiffMedium:
		return combos[0]
	case DiffHard:
		return pickSmartOpening(hand, combos)
	}
	return combos[0]
}

// pickSmartOpening for hard bots: play lowest combo, but prefer sequences
// to break up fewer pairs/triples.
func pickSmartOpening(hand []models.Card, combos []*Play) *Play {
	for _, c := range combos {
		if c.ComboType == models.ComboSequence || c.ComboType == models.ComboDoubleSequence {
			return c
		}
	}
	return combos[0]
}

// chooseBeat finds the best play to beat the current table.
func chooseBeat(hand []models.Card, table *TableState, diff Difficulty) *Play {
	candidates := findBeatingPlays(hand, table)
	if len(candidates) == 0 {
		return nil // pass
	}

	sort.Slice(candidates, func(i, j int) bool {
		return comboStrength(candidates[i]) < comboStrength(candidates[j])
	})

	switch diff {
	case DiffEasy:
		return candidates[rand.Intn(len(candidates))]
	case DiffMedium:
		return candidates[0]
	case DiffHard:
		return pickSmartBeat(hand, candidates, table)
	}
	return candidates[0]
}

func pickSmartBeat(hand []models.Card, candidates []*Play, table *TableState) *Play {
	if len(hand) <= 3 {
		return candidates[len(candidates)-1]
	}

	for _, c := range candidates {
		has2 := false
		for _, card := range c.Cards {
			if card.Rank == models.Two {
				has2 = true
				break
			}
		}
		if !has2 {
			return c
		}
	}
	if len(hand) <= 5 {
		return candidates[0]
	}
	return nil // pass to save 2s
}

// findBeatingPlays enumerates all subsets of hand that can beat the table.
func findBeatingPlays(hand []models.Card, table *TableState) []*Play {
	var results []*Play
	tablePlay := &models.TablePlay{Cards: table.Cards, ComboType: table.ComboType}

	switch table.ComboType {
	case models.ComboSingle:
		for _, c := range hand {
			cards := []models.Card{c}
			if game.CanBeat(tablePlay, cards, models.ComboSingle) {
				results = append(results, &Play{Cards: cards, ComboType: models.ComboSingle})
			}
		}
		if table.Cards[0].Rank == models.Two {
			results = append(results, findChopPlays(hand)...)
		}
	case models.ComboPair:
		pairs := findPairs(hand)
		for _, p := range pairs {
			if game.CanBeat(tablePlay, p, models.ComboPair) {
				results = append(results, &Play{Cards: p, ComboType: models.ComboPair})
			}
		}
		if table.Cards[0].Rank == models.Two {
			for _, ds := range findDoubleSequences(hand) {
				if len(ds) >= 8 {
					results = append(results, &Play{Cards: ds, ComboType: models.ComboDoubleSequence})
				}
			}
		}
	case models.ComboTriple:
		triples := findTriples(hand)
		for _, t := range triples {
			if game.CanBeat(tablePlay, t, models.ComboTriple) {
				results = append(results, &Play{Cards: t, ComboType: models.ComboTriple})
			}
		}
	case models.ComboSequence:
		seqs := findSequences(hand, len(table.Cards))
		for _, s := range seqs {
			if game.CanBeat(tablePlay, s, models.ComboSequence) {
				results = append(results, &Play{Cards: s, ComboType: models.ComboSequence})
			}
		}
	case models.ComboDoubleSequence:
		dseqs := findDoubleSequences(hand)
		for _, ds := range dseqs {
			if len(ds) == len(table.Cards) && game.CanBeat(tablePlay, ds, models.ComboDoubleSequence) {
				results = append(results, &Play{Cards: ds, ComboType: models.ComboDoubleSequence})
			}
		}
	case models.ComboFourOfAKind:
		fours := findFourOfAKinds(hand)
		for _, f := range fours {
			if game.CanBeat(tablePlay, f, models.ComboFourOfAKind) {
				results = append(results, &Play{Cards: f, ComboType: models.ComboFourOfAKind})
			}
		}
	}

	return results
}

func findChopPlays(hand []models.Card) []*Play {
	var results []*Play
	fours := findFourOfAKinds(hand)
	for _, f := range fours {
		results = append(results, &Play{Cards: f, ComboType: models.ComboFourOfAKind})
	}
	dseqs := findDoubleSequences(hand)
	for _, ds := range dseqs {
		if len(ds) >= 6 {
			results = append(results, &Play{Cards: ds, ComboType: models.ComboDoubleSequence})
		}
	}
	return results
}

// decomposeHand breaks a hand into playable combinations, preferring larger combos.
func decomposeHand(hand []models.Card) []*Play {
	models.SortCards(hand)
	var plays []*Play

	used := make([]bool, len(hand))

	seqs := findSequences(hand, 3)
	sort.Slice(seqs, func(i, j int) bool { return len(seqs[i]) > len(seqs[j]) })
	for _, seq := range seqs {
		if canUse(hand, seq, used) {
			markUsed(hand, seq, used)
			ct, _ := game.ClassifyCombination(seq)
			plays = append(plays, &Play{Cards: seq, ComboType: ct})
		}
	}

	dseqs := findDoubleSequences(hand)
	for _, ds := range dseqs {
		if canUse(hand, ds, used) {
			markUsed(hand, ds, used)
			ct, _ := game.ClassifyCombination(ds)
			plays = append(plays, &Play{Cards: ds, ComboType: ct})
		}
	}

	remaining := unusedCards(hand, used)
	byRank := groupByRank(remaining)
	for _, group := range byRank {
		switch len(group) {
		case 4:
			plays = append(plays, &Play{Cards: group, ComboType: models.ComboFourOfAKind})
		case 3:
			plays = append(plays, &Play{Cards: group, ComboType: models.ComboTriple})
		case 2:
			plays = append(plays, &Play{Cards: group, ComboType: models.ComboPair})
		case 1:
			plays = append(plays, &Play{Cards: group, ComboType: models.ComboSingle})
		}
	}

	return plays
}

func lowestCard(hand []models.Card) models.Card {
	models.SortCards(hand)
	return hand[0]
}

func comboStrength(p *Play) int {
	models.SortCards(p.Cards)
	return p.Cards[len(p.Cards)-1].Value()
}

// --- Combination finders ---

func findPairs(hand []models.Card) [][]models.Card {
	byRank := groupByRank(hand)
	var pairs [][]models.Card
	for _, group := range byRank {
		if len(group) >= 2 {
			for i := 0; i < len(group); i++ {
				for j := i + 1; j < len(group); j++ {
					pairs = append(pairs, []models.Card{group[i], group[j]})
				}
			}
		}
	}
	return pairs
}

func findTriples(hand []models.Card) [][]models.Card {
	byRank := groupByRank(hand)
	var triples [][]models.Card
	for _, group := range byRank {
		if len(group) >= 3 {
			for i := 0; i < len(group); i++ {
				for j := i + 1; j < len(group); j++ {
					for k := j + 1; k < len(group); k++ {
						triples = append(triples, []models.Card{group[i], group[j], group[k]})
					}
				}
			}
		}
	}
	return triples
}

func findFourOfAKinds(hand []models.Card) [][]models.Card {
	byRank := groupByRank(hand)
	var fours [][]models.Card
	for _, group := range byRank {
		if len(group) == 4 {
			fours = append(fours, group)
		}
	}
	return fours
}

func findSequences(hand []models.Card, minLen int) [][]models.Card {
	models.SortCards(hand)
	var results [][]models.Card

	ranks := distinctRanks(hand)
	for start := 0; start < len(ranks); start++ {
		if ranks[start] == models.Two {
			continue
		}
		seq := []models.Rank{ranks[start]}
		for next := start + 1; next < len(ranks); next++ {
			if ranks[next] == models.Two {
				break
			}
			if ranks[next] == seq[len(seq)-1]+1 {
				seq = append(seq, ranks[next])
			} else {
				break
			}
		}
		if len(seq) >= minLen {
			for endLen := minLen; endLen <= len(seq); endLen++ {
				subSeq := seq[:endLen]
				cards := pickOnePerRank(hand, subSeq)
				if len(cards) == endLen {
					results = append(results, cards)
				}
			}
		}
	}
	return results
}

func findDoubleSequences(hand []models.Card) [][]models.Card {
	models.SortCards(hand)
	byRank := groupByRank(hand)
	var pairRanks []models.Rank
	for _, r := range distinctRanks(hand) {
		if r == models.Two {
			continue
		}
		if len(byRank[r]) >= 2 {
			pairRanks = append(pairRanks, r)
		}
	}

	var results [][]models.Card
	for start := 0; start < len(pairRanks); start++ {
		consec := []models.Rank{pairRanks[start]}
		for next := start + 1; next < len(pairRanks); next++ {
			if pairRanks[next] == consec[len(consec)-1]+1 {
				consec = append(consec, pairRanks[next])
			} else {
				break
			}
		}
		if len(consec) >= 3 {
			for endLen := 3; endLen <= len(consec); endLen++ {
				subSeq := consec[:endLen]
				var cards []models.Card
				for _, r := range subSeq {
					g := byRank[r]
					cards = append(cards, g[0], g[1])
				}
				results = append(results, cards)
			}
		}
	}
	return results
}

// --- Helpers ---

func groupByRank(cards []models.Card) map[models.Rank][]models.Card {
	m := make(map[models.Rank][]models.Card)
	for _, c := range cards {
		m[c.Rank] = append(m[c.Rank], c)
	}
	return m
}

func distinctRanks(cards []models.Card) []models.Rank {
	seen := make(map[models.Rank]bool)
	var ranks []models.Rank
	for _, c := range cards {
		if !seen[c.Rank] {
			seen[c.Rank] = true
			ranks = append(ranks, c.Rank)
		}
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })
	return ranks
}

func pickOnePerRank(hand []models.Card, ranks []models.Rank) []models.Card {
	byRank := groupByRank(hand)
	var result []models.Card
	for _, r := range ranks {
		if g, ok := byRank[r]; ok && len(g) > 0 {
			result = append(result, g[0])
		}
	}
	return result
}

func canUse(hand []models.Card, cards []models.Card, used []bool) bool {
	needed := make(map[int]int)
	for _, c := range cards {
		needed[c.Value()]++
	}
	for i, c := range hand {
		if used[i] {
			continue
		}
		v := c.Value()
		if needed[v] > 0 {
			needed[v]--
		}
	}
	for _, n := range needed {
		if n > 0 {
			return false
		}
	}
	return true
}

func markUsed(hand []models.Card, cards []models.Card, used []bool) {
	needed := make(map[int]int)
	for _, c := range cards {
		needed[c.Value()]++
	}
	for i, c := range hand {
		if used[i] {
			continue
		}
		v := c.Value()
		if needed[v] > 0 {
			used[i] = true
			needed[v]--
		}
	}
}

func unusedCards(hand []models.Card, used []bool) []models.Card {
	var result []models.Card
	for i, c := range hand {
		if !used[i] {
			result = append(result, c)
		}
	}
	return result
}
