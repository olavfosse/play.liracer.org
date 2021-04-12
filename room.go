package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type room struct {
	sync.Mutex
	// players is the set of players currently in Room.
	players map[*player]struct{}
	snippet string
	roundId roundId
}

type roundId int

// newRoom creates a new room with a random snippet.
func newRoom() *room {
	return &room{
		players: make(map[*player]struct{}),
		snippet: randomSnippet(),
		roundId: 1,
	}
}

// CONCURRENCY_UNSAFE_sendToAllExcept sends bs to all players in r except p.
// CONCURRENCY_UNSAFE_sendToAllExcept is not concurrency safe.
func (r *room) CONCURRENCY_UNSAFE_sendToAllExcept(p *player, bs []byte) {
	for pp := range r.players {
		if p != pp {
			pp.WriteMessage(websocket.TextMessage, bs)
		}
	}
	log.Printf("wrote to all players except %v: %q\n", p, bs)
}

// CONCURRENCY_UNSAFE_sendToAll sends bs to all players in r.
// CONCURRENCY_UNSAFE_sendToAll is not concurrency-safe.
func (r *room) CONCURRENCY_UNSAFE_sendToAll(bs []byte) {
	for pp := range r.players {
		pp.WriteMessage(websocket.TextMessage, bs)
	}
	log.Printf("wrote to all players: %q\n", bs)
}

func (r *room) handlePlayerTypedCorrectChars(p *player, correctChars int) {
	r.Lock()
	defer r.Unlock()
	if correctChars == len(r.snippet) {
		r.snippet = randomSnippet()
		oldId := r.roundId
		r.roundId++
		bs, err := json.Marshal(outgoingMsg{
			NewRoundMsg: &NewRoundOutgoingMsg{
				Snippet:    r.snippet,
				NewRoundId: r.roundId,
				RoundId:    oldId,
			},
		})
		if err != nil {
			log.Println("error:", err)
			panic("marshalling a outgoingMsg should never result in an error")
		}
		r.CONCURRENCY_UNSAFE_sendToAll(bs)
		return
	}
	bs, err := json.Marshal(
		outgoingMsg{
			OpponentCorrectCharsMsg: &OpponentCorrectCharsIncomingMsg{
				OpponentID:   p.id,
				CorrectChars: correctChars,
				RoundId:      r.roundId,
			},
		},
	)
	if err != nil {
		log.Println("error:", err)
		panic("marshalling a outgoingMsg should never result in an error")
	}
	r.CONCURRENCY_UNSAFE_sendToAllExcept(p, bs)
}

func (r *room) handlePlayerJoined(p *player) {
	r.Lock()
	defer r.Unlock()

	r.players[p] = struct{}{}
	log.Printf("registered %v to room, there are now %d players in the room\n", p, len(r.players))

	bs, err := json.Marshal(
		outgoingMsg{
			NewRoundMsg: &NewRoundOutgoingMsg{
				Snippet:    r.snippet,
				NewRoundId: r.roundId,
				RoundId:    0,
			},
		},
	)
	if err != nil {
		log.Println("error:", err)
		panic("marshalling a outgoingMsg should never result in an error")
	}
	p.WriteMessage(websocket.TextMessage, bs)
	log.Printf("wrote: %q\n", bs)
}