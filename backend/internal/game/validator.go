package game

import (
	"github.com/game-playzui/tienlen-server/internal/models"
)

// ClassifyCombination determines the type of a set of cards.
// Returns the combination type and whether it's valid.
func ClassifyCombination(cards []models.Card) (models.CombinationType, bool) {
	models.SortCards(cards)
	n := len(cards)

	switch {
	case n == 1:
		return models.ComboSingle, true
	case n == 2:
		return classifyPairOrInvalid(cards)
	case n == 3:
		return classifyTripleOrSequence(cards)
	case n == 4:
		return classifyFour(cards)
	case n >= 3:
		return classifyLargeCombo(cards)
	}
	return 0, false
}

func classifyPairOrInvalid(cards []models.Card) (models.CombinationType, bool) {
	if cards[0].Rank == cards[1].Rank {
		return models.ComboPair, true
	}
	return 0, false
}

func classifyTripleOrSequence(cards []models.Card) (models.CombinationType, bool) {
	if cards[0].Rank == cards[1].Rank && cards[1].Rank == cards[2].Rank {
		return models.ComboTriple, true
	}
	if isSequence(cards) {
		return models.ComboSequence, true
	}
	return 0, false
}

func classifyFour(cards []models.Card) (models.CombinationType, bool) {
	if cards[0].Rank == cards[1].Rank && cards[1].Rank == cards[2].Rank && cards[2].Rank == cards[3].Rank {
		return models.ComboFourOfAKind, true
	}
	if isSequence(cards) {
		return models.ComboSequence, true
	}
	if isDoubleSequence(cards) {
		return models.ComboDoubleSequence, true
	}
	return 0, false
}

func classifyLargeCombo(cards []models.Card) (models.CombinationType, bool) {
	if isSequence(cards) {
		return models.ComboSequence, true
	}
	if isDoubleSequence(cards) {
		return models.ComboDoubleSequence, true
	}
	return 0, false
}

// isSequence checks if cards form a consecutive sequence of singles.
// Sequences cannot contain 2s and must be at least 3 cards.
func isSequence(cards []models.Card) bool {
	if len(cards) < 3 {
		return false
	}
	for _, c := range cards {
		if c.Rank == models.Two {
			return false
		}
	}
	for i := 1; i < len(cards); i++ {
		if cards[i].Rank != cards[i-1].Rank+1 {
			return false
		}
	}
	return true
}

// isDoubleSequence checks if cards form consecutive pairs (e.g., 33-44-55).
// Must have even count >= 6, no 2s.
func isDoubleSequence(cards []models.Card) bool {
	n := len(cards)
	if n < 6 || n%2 != 0 {
		return false
	}
	for _, c := range cards {
		if c.Rank == models.Two {
			return false
		}
	}
	for i := 0; i < n; i += 2 {
		if cards[i].Rank != cards[i+1].Rank {
			return false
		}
		if i > 0 && cards[i].Rank != cards[i-2].Rank+1 {
			return false
		}
	}
	return true
}

// CanBeat checks whether 'play' can beat 'table' according to Tien Len rules.
func CanBeat(table *models.TablePlay, play []models.Card, playCombo models.CombinationType) bool {
	if table == nil {
		return true
	}

	tableCards := table.Cards
	tableCombo := table.ComboType

	// "Chop pig" rules: four-of-a-kind or double sequence of 3+ pairs beats a single 2
	if tableCombo == models.ComboSingle && tableCards[0].Rank == models.Two {
		if playCombo == models.ComboFourOfAKind {
			return true
		}
		if playCombo == models.ComboDoubleSequence && len(play) >= 6 {
			return true
		}
	}

	// "Chop" pair of 2s with double sequence of 4+ pairs
	if tableCombo == models.ComboPair && tableCards[0].Rank == models.Two {
		if playCombo == models.ComboDoubleSequence && len(play) >= 8 {
			return true
		}
	}

	// Normal beating: same combo type, same card count, higher value
	if playCombo != tableCombo || len(play) != len(tableCards) {
		return false
	}

	models.SortCards(play)
	tableSorted := make([]models.Card, len(tableCards))
	copy(tableSorted, tableCards)
	models.SortCards(tableSorted)

	// Compare the highest card in each combination
	playHigh := play[len(play)-1]
	tableHigh := tableSorted[len(tableSorted)-1]

	return playHigh.Value() > tableHigh.Value()
}

// PlayerOwnsCards checks that all cards in 'played' exist in 'hand'
func PlayerOwnsCards(hand, played []models.Card) bool {
	handMap := make(map[int]int)
	for _, c := range hand {
		handMap[c.Value()]++
	}
	for _, c := range played {
		v := c.Value()
		if handMap[v] <= 0 {
			return false
		}
		handMap[v]--
	}
	return true
}
