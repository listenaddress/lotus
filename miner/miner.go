package miner

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	chain "github.com/filecoin-project/go-lotus/chain"
)

var log = logging.Logger("miner")

type api interface {
	SubmitNewBlock(blk *chain.BlockMsg) error

	// returns a set of messages that havent been included in the chain as of
	// the given tipset
	PendingMessages(base *chain.TipSet) ([]*chain.SignedMessage, error)

	// Returns the best tipset for the miner to mine on top of.
	// TODO: Not sure this feels right (including the messages api). Miners
	// will likely want to have more control over exactly which blocks get
	// mined on, and which messages are included.
	GetBestTipset() (*chain.TipSet, error)

	LatestTipsets() (<-chan *chain.TipSet, error)

	// returns the lookback randomness from the chain used for the election
	GetChainRandomness(ts *chain.TipSet) ([]byte, error)

	// create a block
	// it seems realllllly annoying to do all the actions necessary to build a
	// block through the API. so, we just add the block creation to the API
	// now, all the 'miner' does is check if they win, and call create block
	CreateBlock(base *chain.TipSet, tickets []chain.Ticket, eproof chain.ElectionProof, msgs []*chain.SignedMessage) (*chain.BlockMsg, error)
}

type Miner struct {
	api api

	// time between blocks, network parameter
	Delay time.Duration

	lastWork *MiningBase
}

func (m *Miner) Mine(ctx context.Context) {
	tsetch, err := m.api.LatestTipsets()
	if err != nil {
		log.Errorf("failed to get latest tipset channel, shutting down: %s", err)
		return
	}

	best, ok := <-tsetch
	if !ok {
		log.Errorf("couldnt get initial best tipset, exiting mining process")
		return
	}

	cur := &MiningBase{
		ts: best,
	}

	miningDone := make(chan *chain.BlockMsg)

	go func() {
		b, err := m.mineOne(ctx, cur)
		if err != nil {
			log.Errorf("mining block failed: %s", err)
			return
		}

		miningDone <- b
	}()

	for {
		select {
		case ts, ok := <-tsetch:
			if !ok {
				log.Error("tipset channel closed, exiting mining process")
				return
			}

			if ts.Weight() > cur.ts.Weight() {
				cur = &MiningBase{ts: ts}
			}
			// Interrupt running mining job?

		case blk := <-miningDone:
			if err := m.api.SubmitNewBlock(blk); err != nil {
				log.Errorf("failed to submit newly mined block: %s", err)
			}

			go func() {
				b, err := m.mineOne(ctx, cur)
				if err != nil {
					log.Errorf("mining block failed: %s", err)
					return
				}

				miningDone <- b
			}()
		}
	}

	/*
			base, err := m.GetBestMiningCandidate()
			if err != nil {
				log.Errorf("failed to get best mining candidate: %s", err)
				continue
			}

			b, err := m.mineOne(ctx, base)
			if err != nil {
				log.Errorf("mining block failed: %s", err)
				continue
			}

			if b != nil {
				if err := m.api.SubmitNewBlock(b); err != nil {
					log.Errorf("failed to submit newly mined block: %s", err)
				}
			}
		}
	*/
}

type MiningBase struct {
	ts      *chain.TipSet
	tickets []chain.Ticket
}

func (m *Miner) GetBestMiningCandidate() (*MiningBase, error) {
	bts, err := m.api.GetBestTipset()
	if err != nil {
		return nil, err
	}

	if m.lastWork != nil {
		if m.lastWork.ts.Equals(bts) {
			return m.lastWork, nil
		}

		if bts.Weight() <= m.lastWork.ts.Weight() {
			return m.lastWork, nil
		}
	}

	return &MiningBase{
		ts: bts,
	}, nil
}

func (m *Miner) mineOne(ctx context.Context, base *MiningBase) (*chain.BlockMsg, error) {
	log.Info("mine one")
	ticket, err := m.scratchTicket(ctx, base)
	if err != nil {
		return nil, errors.Wrap(err, "scratching ticket failed")
	}

	win, proof, err := m.isWinnerNextRound(base)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if we win next round")
	}

	if !win {
		m.submitNullTicket(base, ticket)
		return nil, nil
	}

	b, err := m.createBlock(base, ticket, proof)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create block")
	}
	log.Infof("created new block: %s", b.Cid())

	return b, nil
}

func (m *Miner) submitNullTicket(base *MiningBase, ticket chain.Ticket) {
	base.tickets = append(base.tickets, ticket)
	m.lastWork = base
}

func (m *Miner) isWinnerNextRound(base *MiningBase) (bool, chain.ElectionProof, error) {
	r, err := m.api.GetChainRandomness(base.ts)
	if err != nil {
		return false, nil, err
	}

	_ = r // TODO: use this to properly compute the election proof

	return true, []byte("election prooooof"), nil
}

func (m *Miner) scratchTicket(ctx context.Context, base *MiningBase) (chain.Ticket, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.Delay):
	}

	return []byte("this is a ticket"), nil
}

func (m *Miner) createBlock(base *MiningBase, ticket chain.Ticket, proof chain.ElectionProof) (*chain.BlockMsg, error) {

	pending, err := m.api.PendingMessages(base.ts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pending messages")
	}

	// why even return this? that api call could just submit it for us
	return m.api.CreateBlock(base.ts, append(base.tickets, ticket), proof, pending)
}

func (m *Miner) selectMessages(msgs []*chain.SignedMessage) []*chain.SignedMessage {
	// TODO: filter and select 'best' message if too many to fit in one block
	return msgs
}