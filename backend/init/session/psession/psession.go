package psession

import (
	"encoding/json"
	"time"

	"github.com/1Panel-dev/1Panel/backend/init/cache/badger_db"
)

type SessionUser struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type PSession struct {
	ExpireTime int64 `json:"expire_time"`
	store      *badger_db.Cache
}

func NewPSession(db *badger_db.Cache) *PSession {
	return &PSession{
		store: db,
	}
}

func (p *PSession) Get(sessionID string) (SessionUser, error) {
	var result SessionUser
	item, err := p.store.Get(sessionID)
	if err != nil {
		return result, err
	}
	_ = json.Unmarshal(item, &result)
	return result, nil
}

func (p *PSession) Set(sessionID string, user SessionUser, ttlSeconds int) error {
	// The underlying cache serializes values with %v, which mangles structs;
	// marshal to JSON so Get's json.Unmarshal round-trips the user identity.
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	p.ExpireTime = time.Now().Unix() + int64(ttlSeconds)
	return p.store.SetWithTTL(sessionID, string(data), time.Second*time.Duration(ttlSeconds))
}

func (p *PSession) Delete(sessionID string) error {
	return p.store.Del(sessionID)
}

func (p *PSession) Clean() error {
	return p.store.Clean()
}
