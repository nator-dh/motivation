package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	configPath string
	notifier   *Notifier

	mu      sync.RWMutex
	cfg     *Config
	cron    *cron.Cron
	loc     *time.Location
	entries []entryInfo // for /api/state
}

type entryInfo struct {
	Kind     string       `json:"kind"` // "quote" or "general"
	CronExpr string       `json:"cron"`
	QuoteID  string       `json:"quote_id,omitempty"`
	Category string       `json:"category,omitempty"`
	NextRun  time.Time    `json:"next_run"`
	id       cron.EntryID `json:"-"`
}

func NewScheduler(configPath string, n *Notifier) *Scheduler {
	return &Scheduler{configPath: configPath, notifier: n}
}

func (s *Scheduler) Start() error {
	return s.Reload()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}
}

func (s *Scheduler) Reload() error {
	cfg, err := LoadConfig(s.configPath)
	if err != nil {
		return err
	}

	loc := time.Local
	if cfg.Defaults.Timezone != "" {
		l, err := time.LoadLocation(cfg.Defaults.Timezone)
		if err != nil {
			return fmt.Errorf("load timezone %q: %w", cfg.Defaults.Timezone, err)
		}
		loc = l
	}

	c := cron.New(cron.WithLocation(loc))
	var entries []entryInfo

	for _, q := range cfg.Quotes {
		for _, expr := range q.Schedules {
			id, err := c.AddFunc(expr, func() { s.fireQuote(q) })
			if err != nil {
				return fmt.Errorf("quote %s cron %q: %w", q.ID, expr, err)
			}
			entries = append(entries, entryInfo{
				Kind: "quote", CronExpr: expr, QuoteID: q.ID, id: id,
			})
		}
	}

	for _, gs := range cfg.GeneralSchedules {
		id, err := c.AddFunc(gs.Cron, func() { s.fireRandom(gs.Category) })
		if err != nil {
			return fmt.Errorf("general schedule %q: %w", gs.Cron, err)
		}
		entries = append(entries, entryInfo{
			Kind: "general", CronExpr: gs.Cron, Category: gs.Category, id: id,
		})
	}

	c.Start()

	s.mu.Lock()
	old := s.cron
	s.cfg = cfg
	s.cron = c
	s.loc = loc
	s.entries = entries
	s.mu.Unlock()

	if old != nil {
		ctx := old.Stop()
		<-ctx.Done()
	}
	log.Printf("scheduler: loaded %d quotes, %d general schedules, tz=%s",
		len(cfg.Quotes), len(cfg.GeneralSchedules), loc)
	return nil
}

func (s *Scheduler) fireQuote(q Quote) {
	log.Printf("fire quote %s (%s)", q.ID, q.Author)
	var openURL string
	if len(q.Media) > 0 {
		openURL = q.Media[0]
	}
	if err := s.notifier.Notify(q.Author, q.Text, openURL); err != nil {
		log.Printf("notify error: %v", err)
	}
}

func (s *Scheduler) fireRandom(category string) {
	s.mu.RLock()
	pool := make([]Quote, 0, len(s.cfg.Quotes))
	for _, q := range s.cfg.Quotes {
		if q.HasCategory(category) {
			pool = append(pool, q)
		}
	}
	s.mu.RUnlock()
	if len(pool) == 0 {
		log.Printf("general schedule (category=%q) matched no quotes", category)
		return
	}
	q := pool[rand.IntN(len(pool))]
	s.fireQuote(q)
}

func (s *Scheduler) FireByID(id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, q := range s.cfg.Quotes {
		if q.ID == id {
			go s.fireQuote(q)
			return nil
		}
	}
	return fmt.Errorf("quote %q not found", id)
}

func (s *Scheduler) Snapshot() (*Config, []entryInfo) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfgCopy := *s.cfg
	entries := make([]entryInfo, len(s.entries))
	copy(entries, s.entries)
	if s.cron != nil {
		for i, e := range entries {
			entries[i].NextRun = s.cron.Entry(e.id).Next
		}
	}
	return &cfgCopy, entries
}
