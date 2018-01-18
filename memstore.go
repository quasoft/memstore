package memstore

import (
	"bytes"
	"encoding/gob"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// MemStore is an in-memory implementation of gorilla/sessions, suitable
// for use in tests and development environments. Do not use in production.
// Values are cached in a map. The cache is protected and can be used by
// multiple goroutines.
type MemStore struct {
	Codecs        []securecookie.Codec
	Options       *sessions.Options
	DefaultMaxAge int
	cache         *cache
}

type valueType = map[interface{}]interface{}

// NewMemStore returns a new MemStore.
//
// Keys are defined in pairs to allow key rotation, but the common case is
// to set a single authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for
// encryption. The encryption key can be set to nil or omitted in the last
// pair, but the authentication key is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes.
// The encryption key, if set, must be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256 modes.
//
// Use the convenience function securecookie.GenerateRandomKey() to create
// strong keys.
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

// Get returns a session for the given name after adding it to the registry.
//
// It returns a new session if the sessions doesn't exist. Access IsNew on
// the session to check if it is an existing session or a new one.
//
// It returns a new session and an error if the session exists but could
// not be decoded.
func (m *MemStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

// New returns a session for the given name without adding it to the registry.
//
// The difference between New() and Get() is that calling New() twice will
// decode the session data twice, while Get() registers and reuses the same
// decoded session after the first call.
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

// Save adds a single session to the response.
// Set Options.MaxAge to -1 before saving the session to delete all values in it.
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

func (m *MemStore) copy(v valueType) (valueType, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	var values valueType
	err = dec.Decode(&values)
	if err != nil {
		return nil, err
	}
	return values, nil
}
