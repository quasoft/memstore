package memstore

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/sessions"
)

func TestMemStore_Get(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	req := httptest.NewRequest("GET", "http://test.local", nil)
	_, err := store.Get(req, "mycookiename")
	if err != nil {
		t.Errorf("failed to create session from empty request: %v", err)
	}
}

func TestMemStore_Get_Bogus(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	req := httptest.NewRequest("GET", "http://test.local", nil)
	req.AddCookie(sessions.NewCookie("mycookiename", "SomeBogusValueThatIsActuallyNotEncrypted", store.Options))
	_, err := store.Get(req, "mycookiename")
	if err == nil {
		t.Error(`store.Get(req, "mycookiename") should have returned error if cookie value is bogus`)
	}
}

func TestMemStore_New(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	req := httptest.NewRequest("GET", "http://test.local", nil)
	_, err := store.New(req, "mycookiename")
	if err != nil {
		t.Errorf("failed to create session from empty request: %v", err)
	}
}

func TestMemStore_Save(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	want := "value123"
	req := httptest.NewRequest("GET", "http://test.local", nil)
	rec := httptest.NewRecorder()
	session, err := store.Get(req, "mycookiename")
	if err != nil {
		t.Fatalf("failed to create session from empty request: %v", err)
	}
	session.Values["key"] = want
	session.Save(req, rec)

	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "mycookiename") {
		t.Error("cookie was not stored in request")
	}
}

func TestMemStore_Save_Multiple_Requests(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	want := "value123"

	req1 := httptest.NewRequest("GET", "http://test.local", nil)
	rec1 := httptest.NewRecorder()
	session1, err := store.Get(req1, "mycookiename")
	if err != nil {
		t.Fatalf("failed to create session from empty request: %v", err)
	}
	session1.Values["key"] = want
	session1.Save(req1, rec1)

	req2 := httptest.NewRequest("GET", "http://test.local", nil)
	// Simulate retaining cookie from previous response
	req2.AddCookie(rec1.Result().Cookies()[0])
	session2, err := store.Get(req2, "mycookiename")
	if err != nil {
		t.Fatalf("failed to create session from second request: %v", err)
	}
	got := session2.Values["key"]
	if got != want {
		t.Errorf(`session2.Values["key"] got = %q, want %q`, got, want)
	}
}

func TestMemStore_Delete(t *testing.T) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	req := httptest.NewRequest("GET", "http://test.local", nil)
	rec := httptest.NewRecorder()
	session, err := store.Get(req, "mycookiename")
	if err != nil {
		t.Fatalf("failed to create session from empty request: %v", err)
	}

	// Save some value
	session.Values["key"] = "somevalue"
	session.Save(req, rec)

	// And immediately delete it
	session.Options.MaxAge = -1
	session.Save(req, rec)

	if session.Values["key"] == "somevalue" {
		t.Error("cookie was not deleted from session after setting session.Options.MaxAge = -1 and saving")
	}
}

func BenchmarkRace(b *testing.B) {
	store := NewMemStore(
		[]byte("authkey"),
		[]byte("enckey1234567890"),
	)

	var wg sync.WaitGroup

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()

		session, err := store.Get(r, "mycookiename")
		if err != nil {
			b.Fatalf("failed to create session from empty request: %v", err)
		}
		session.Values["key"] = "somevalue"
		session.Save(r, w)

		// And immediately delete it
		session.Options.MaxAge = -1
		session.Save(r, w)

		_ = session.Values["key"]
	}))
	defer s.Close()

	loops := 100
	wg.Add(loops)
	for i := 1; i <= loops; i++ {
		go http.Get(s.URL)
	}
	wg.Wait()
}
