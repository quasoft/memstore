package memstore

import (
	"bytes"
	"encoding/gob"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// MemStore is an in-memory implementation of gorilla/sessions suitable
// for use in tests and development environments. Do not use in production.
type MemStore struct {
	Codecs        []securecookie.Codec
	Options       *sessions.Options
	DefaultMaxAge int
	cache         cache
}

type ValueType = map[interface{}]interface{}

func NewMemStore(keyPairs ...[]byte) *MemStore {
	store := MemStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path: "/",
		},
		cache: newCache(),
	}
	return &store
}

func (m *MemStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

func (m *MemStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var err error
	session := sessions.NewSession(m, name)
	options := *m.Options
	session.Options = &options
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err := securecookie.DecodeMulti(name, c.Value, &session.ID, m.Codecs...)
		if err == nil {
			v, ok := m.cache.value(name)
			if ok {
				values, err := m.copy(v)
				if err == nil {
					session.Values = values
				}
			}
			session.IsNew = !ok
		}
	}
	return session, err
}

func (m *MemStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	if s.Options.MaxAge < 0 {
		m.cache.delete(s.Name())
		http.SetCookie(w, sessions.NewCookie(s.Name(), "", s.Options))
		for k := range s.Values {
			delete(s.Values, k)
		}
	} else {
		sessionValues, err := m.copy(s.Values)
		if err != nil {
			return err
		}
		m.cache.setValue(s.Name(), sessionValues)

		encoded, err := securecookie.EncodeMulti(s.Name(), s.ID, m.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(s.Name(), encoded, s.Options))
	}
	return nil
}

func (m *MemStore) copy(v ValueType) (ValueType, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	var values ValueType
	err = dec.Decode(&values)
	if err != nil {
		return nil, err
	}
	return values, nil
}
