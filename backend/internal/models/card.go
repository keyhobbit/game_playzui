package models

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
)

type Suit int

const (
	Spades   Suit = iota // lowest
	Clubs
	Diamonds
	Hearts // highest
)

func (s Suit) String() string {
	return [...]string{"S", "C", "D", "H"}[s]
}

func (s Suit) Name() string {
	return [...]string{"Spades", "Clubs", "Diamonds", "Hearts"}[s]
}

func ParseSuit(s string) (Suit, error) {
	switch s {
	case "S":
		return Spades, nil
	case "C":
		return Clubs, nil
	case "D":
		return Diamonds, nil
	case "H":
		return Hearts, nil
	}
	return 0, fmt.Errorf("invalid suit: %s", s)
}

type Rank int

const (
	Three Rank = iota
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
	Ace
	Two // highest
)

func (r Rank) String() string {
	return [...]string{"3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A", "2"}[r]
}

func ParseRank(s string) (Rank, error) {
	switch s {
	case "3":
		return Three, nil
	case "4":
		return Four, nil
	case "5":
		return Five, nil
	case "6":
		return Six, nil
	case "7":
		return Seven, nil
	case "8":
		return Eight, nil
	case "9":
		return Nine, nil
	case "10":
		return Ten, nil
	case "J":
		return Jack, nil
	case "Q":
		return Queen, nil
	case "K":
		return King, nil
	case "A":
		return Ace, nil
	case "2":
		return Two, nil
	}
	return 0, fmt.Errorf("invalid rank: %s", s)
}

type Card struct {
	Rank Rank `json:"rank"`
	Suit Suit `json:"suit"`
}

type cardJSON struct {
	Rank string `json:"rank"`
	Suit string `json:"suit"`
}

func (c Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(cardJSON{Rank: c.Rank.String(), Suit: c.Suit.String()})
}

func (c *Card) UnmarshalJSON(data []byte) error {
	var raw cardJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var err error
	c.Rank, err = ParseRank(raw.Rank)
	if err != nil {
		return err
	}
	c.Suit, err = ParseSuit(raw.Suit)
	return err
}

// Value returns a single integer for comparison: rank * 4 + suit
func (c Card) Value() int {
	return int(c.Rank)*4 + int(c.Suit)
}

func CompareCards(a, b Card) int {
	return a.Value() - b.Value()
}

func SortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Value() < cards[j].Value()
	})
}

func NewDeck() []Card {
	deck := make([]Card, 0, 52)
	for r := Three; r <= Two; r++ {
		for s := Spades; s <= Hearts; s++ {
			deck = append(deck, Card{Rank: r, Suit: s})
		}
	}
	return deck
}

func ShuffleDeck(deck []Card) {
	n := len(deck)
	for i := n - 1; i > 0; i-- {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := jBig.Int64()
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func DealCards() [4][]Card {
	deck := NewDeck()
	ShuffleDeck(deck)

	var hands [4][]Card
	for i := 0; i < 4; i++ {
		hands[i] = make([]Card, 13)
		copy(hands[i], deck[i*13:(i+1)*13])
		SortCards(hands[i])
	}
	return hands
}

func ThreeOfSpades() Card {
	return Card{Rank: Three, Suit: Spades}
}

func ContainsCard(hand []Card, card Card) bool {
	for _, c := range hand {
		if c.Rank == card.Rank && c.Suit == card.Suit {
			return true
		}
	}
	return false
}

func RemoveCards(hand []Card, cards []Card) []Card {
	result := make([]Card, 0, len(hand))
	remove := make(map[int]bool)
	for _, c := range cards {
		remove[c.Value()] = true
	}
	for _, c := range hand {
		if !remove[c.Value()] {
			result = append(result, c)
		} else {
			delete(remove, c.Value())
		}
	}
	return result
}
