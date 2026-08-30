package pagination

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	gormtest "gorm.io/gorm/utils/tests"
)

var contextSecret = []byte("integration-secret")

const (
	resourceContext   = "resource-items:resource-123"
	legacyCursor      = "b64!eyJ2IjoxLCJsaW1pdCI6MjUsInNvcnQiOltbImlkIiwiZGVzYyJdXSwiYWZ0ZXIiOnsiaWQiOiJhYmMifSwidHMiOjE3MDAwMDAwMDAsInNpZyI6IkdSa05JQVlJcFp3U1hpLW4wMnYxeGVTRldzYldoVU5VdTVXS3J3RVhMMGMifQ=="
	derivedKeyHex     = "a8219ca95972bc7f2986cd22167c51a17c4369efe3779ebaa4cf4741f01283a6"
	fixedCursorTime   = int64(1700000000)
	concurrentWorkers = 32
)

type contextRecord struct {
	ID string `json:"id"`
}

func TestUnscopedSigningCompatibility(t *testing.T) {
	payload := CursorPayload{
		Version:   1,
		Limit:     25,
		Sort:      []SortExpression{{Field: "id", Direction: DirectionDesc}},
		After:     map[string]string{"id": "abc"},
		Timestamp: fixedCursorTime,
	}

	encoded, err := EncodeCursorSigned(payload, contextSecret)
	if err != nil {
		t.Fatalf("encode legacy cursor: %v", err)
	}
	if encoded != legacyCursor {
		t.Fatalf("legacy cursor changed:\nwant %s\n got %s", legacyCursor, encoded)
	}

	emptyContext, err := EncodeCursorSigned(payload, contextSecret, WithSigningContext(""))
	if err != nil {
		t.Fatalf("encode cursor with empty context: %v", err)
	}
	if emptyContext != legacyCursor {
		t.Fatalf("empty context changed legacy cursor:\nwant %s\n got %s", legacyCursor, emptyContext)
	}

	decoded, err := DecodeCursorSigned(legacyCursor, contextSecret, 0)
	if err != nil {
		t.Fatalf("decode legacy cursor: %v", err)
	}
	if decoded.After["id"] != "abc" {
		t.Fatalf("expected legacy cursor id abc, got %q", decoded.After["id"])
	}
}

func TestSigningContextKeyDerivationIsDeterministic(t *testing.T) {
	first := deriveCursorSigningKey(contextSecret, resourceContext)
	second := deriveCursorSigningKey(contextSecret, resourceContext)

	if !bytes.Equal(first, second) {
		t.Fatal("expected identical inputs to derive identical signing keys")
	}
	if got := hex.EncodeToString(first); got != derivedKeyHex {
		t.Fatalf("derived key changed: want %s, got %s", derivedKeyHex, got)
	}
	if bytes.Equal(first, deriveCursorSigningKey(contextSecret, "resource-items:resource-456")) {
		t.Fatal("expected different contexts to derive different signing keys")
	}
	if !bytes.Equal(contextSecret, deriveCursorSigningKey(contextSecret, "")) {
		t.Fatal("expected empty context to preserve the master signing key")
	}
}

func TestScopedCursorRoundTripAndIsolation(t *testing.T) {
	payload := CursorPayload{
		Version:   1,
		Limit:     25,
		Sort:      []SortExpression{{Field: "id", Direction: DirectionDesc}},
		Filters:   map[string]string{"status_eq": "active"},
		After:     map[string]string{"id": "item-123"},
		Timestamp: time.Now().Unix(),
	}

	encoded, err := EncodeCursorSigned(payload, contextSecret, WithSigningContext(resourceContext))
	if err != nil {
		t.Fatalf("encode scoped cursor: %v", err)
	}

	decoded, err := DecodeCursorSigned(encoded, contextSecret, time.Hour, WithSigningContext(resourceContext))
	if err != nil {
		t.Fatalf("decode scoped cursor: %v", err)
	}
	if decoded.After["id"] != payload.After["id"] {
		t.Fatalf("expected after id %q, got %q", payload.After["id"], decoded.After["id"])
	}
	if got := decoded.Filters["status_eq"]; got != "active" {
		t.Fatalf("expected status filter active, got %q", got)
	}

	raw := decodeCursorJSON(t, encoded)
	if strings.Contains(raw, resourceContext) || strings.Contains(raw, "signing_context") {
		t.Fatalf("signing context leaked into cursor payload: %s", raw)
	}

	tests := []struct {
		name string
		opts []SigningOption
	}{
		{name: "wrong resource id", opts: []SigningOption{WithSigningContext("resource-items:resource-456")}},
		{name: "wrong resource namespace", opts: []SigningOption{WithSigningContext("resource-events:resource-123")}},
		{name: "missing context"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeCursorSigned(encoded, contextSecret, time.Hour, test.opts...); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("expected invalid cursor, got %v", err)
			}
		})
	}

	unscoped, err := EncodeCursorSigned(payload, contextSecret)
	if err != nil {
		t.Fatalf("encode unscoped cursor: %v", err)
	}
	if _, err := DecodeCursorSigned(unscoped, contextSecret, time.Hour, WithSigningContext(resourceContext)); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected scoped validation to reject unscoped cursor, got %v", err)
	}

	lastContextWins, err := EncodeCursorSigned(
		payload,
		contextSecret,
		nil,
		WithSigningContext("resource-items:ignored"),
		WithSigningContext(resourceContext),
	)
	if err != nil {
		t.Fatalf("encode cursor with multiple options: %v", err)
	}
	if _, err := DecodeCursorSigned(lastContextWins, contextSecret, time.Hour, WithSigningContext(resourceContext)); err != nil {
		t.Fatalf("expected the last signing context option to win: %v", err)
	}
}

func TestScopedSignedCursorBuilders(t *testing.T) {
	params := Params{
		Limit:   10,
		Sort:    []SortExpression{{Field: "id", Direction: DirectionAsc}},
		Filters: map[string]string{"status_eq": "active"},
	}
	option := WithSigningContext(resourceContext)

	next, err := BuildNextCursorSigned(params, map[string]string{"id": "item-10"}, contextSecret, option)
	if err != nil {
		t.Fatalf("build scoped next cursor: %v", err)
	}
	nextPayload, err := DecodeCursorSigned(next, contextSecret, time.Hour, option)
	if err != nil {
		t.Fatalf("decode scoped next cursor: %v", err)
	}
	if nextPayload.After["id"] != "item-10" {
		t.Fatalf("expected next boundary item-10, got %q", nextPayload.After["id"])
	}

	prev, err := BuildPrevCursorSigned(params, map[string]string{"id": "item-1"}, contextSecret, option)
	if err != nil {
		t.Fatalf("build scoped previous cursor: %v", err)
	}
	prevPayload, err := DecodeCursorSigned(prev, contextSecret, time.Hour, option)
	if err != nil {
		t.Fatalf("decode scoped previous cursor: %v", err)
	}
	if prevPayload.Before["id"] != "item-1" {
		t.Fatalf("expected previous boundary item-1, got %q", prevPayload.Before["id"])
	}
}

func TestScopedCursorRejectsTampering(t *testing.T) {
	payload := CursorPayload{
		Limit:     25,
		Sort:      []SortExpression{{Field: "id", Direction: DirectionAsc}},
		After:     map[string]string{"id": "item-123"},
		Timestamp: time.Now().Unix(),
	}
	encoded, err := EncodeCursorSigned(payload, contextSecret, WithSigningContext(resourceContext))
	if err != nil {
		t.Fatalf("encode scoped cursor: %v", err)
	}

	raw := decodeCursorJSON(t, encoded)
	var tamperedPayload map[string]any
	if err := json.Unmarshal([]byte(raw), &tamperedPayload); err != nil {
		t.Fatalf("unmarshal cursor: %v", err)
	}
	tamperedPayload["after"] = map[string]string{"id": "item-999"}
	tamperedJSON, err := json.Marshal(tamperedPayload)
	if err != nil {
		t.Fatalf("marshal tampered cursor: %v", err)
	}
	tampered := "b64!" + base64.StdEncoding.EncodeToString(tamperedJSON)

	if _, err := DecodeCursorSigned(tampered, contextSecret, time.Hour, WithSigningContext(resourceContext)); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor, got %v", err)
	}
}

func TestScopedCursorExpiration(t *testing.T) {
	payload := CursorPayload{
		Version:   1,
		Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
	}
	encoded, err := EncodeCursorSigned(payload, contextSecret, WithSigningContext(resourceContext))
	if err != nil {
		t.Fatalf("encode expired scoped cursor: %v", err)
	}

	if _, err := DecodeCursorSigned(encoded, contextSecret, time.Hour, WithSigningContext(resourceContext)); !errors.Is(err, ErrCursorExpired) {
		t.Fatalf("expected expired cursor, got %v", err)
	}
}

func TestScopedParseAndBuildPageRoundTrip(t *testing.T) {
	raw := url.Values{}
	raw.Set("limit", "2")
	raw.Set("sort", "id:asc")
	raw.Set("status_eq", "active")
	option := WithSigningContext(resourceContext)

	params, err := ParseWithSecurity(raw, contextSecret, time.Hour, option)
	if err != nil {
		t.Fatalf("parse first page: %v", err)
	}
	results := []contextRecord{{ID: "item-1"}, {ID: "item-2"}, {ID: "item-3"}}
	page, err := BuildPageSigned(params, results, params.Limit, 3, nil, contextSecret, option)
	if err != nil {
		t.Fatalf("build scoped page: %v", err)
	}
	if page.Paging.NextCursor == "" {
		t.Fatal("expected next cursor")
	}

	nextRaw := url.Values{"cursor": []string{page.Paging.NextCursor}}
	nextParams, err := ParseWithSecurity(nextRaw, contextSecret, time.Hour, option)
	if err != nil {
		t.Fatalf("parse scoped next cursor: %v", err)
	}
	if len(nextParams.Filters) != 1 || nextParams.Filters["status_eq"] != "active" {
		t.Fatalf("expected only application filters, got %#v", nextParams.Filters)
	}

	blockedParams, err := ParseWithSecurityBlockedFilters(nextRaw, contextSecret, time.Hour, []string{"owner_id"}, option)
	if err != nil {
		t.Fatalf("parse scoped cursor with blocked filters: %v", err)
	}
	if blockedParams.Cursor == nil {
		t.Fatal("expected blocked-filter parser to preserve scoped cursor")
	}

	db, err := gorm.Open(gormtest.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}
	if _, err := Apply(db.Model(&contextRecord{}), nextParams); err != nil {
		t.Fatalf("apply parsed scoped params without context: %v", err)
	}
	if _, err := ApplyFilters(db.Model(&contextRecord{}), nextParams.Filters); err != nil {
		t.Fatalf("apply scoped filters without context: %v", err)
	}
}

func TestScopedSigningConcurrentContexts(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, concurrentWorkers)

	for worker := 0; worker < concurrentWorkers; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			context := fmt.Sprintf("resource-events:resource-%d", worker)
			payload := CursorPayload{
				Limit:     10,
				Sort:      []SortExpression{{Field: "id", Direction: DirectionAsc}},
				After:     map[string]string{"id": fmt.Sprintf("event-%d", worker)},
				Timestamp: time.Now().Unix(),
			}

			encoded, err := EncodeCursorSigned(payload, contextSecret, WithSigningContext(context))
			if err != nil {
				errs <- fmt.Errorf("worker %d encode: %w", worker, err)
				return
			}
			decoded, err := DecodeCursorSigned(encoded, contextSecret, time.Hour, WithSigningContext(context))
			if err != nil {
				errs <- fmt.Errorf("worker %d decode: %w", worker, err)
				return
			}
			if decoded.After["id"] != payload.After["id"] {
				errs <- fmt.Errorf("worker %d decoded another context's cursor", worker)
				return
			}

			other := fmt.Sprintf("resource-events:resource-%d", (worker+1)%concurrentWorkers)
			if _, err := DecodeCursorSigned(encoded, contextSecret, time.Hour, WithSigningContext(other)); !errors.Is(err, ErrInvalidCursor) {
				errs <- fmt.Errorf("worker %d cross-context decode: %v", worker, err)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func decodeCursorJSON(t *testing.T, encoded string) string {
	t.Helper()
	if !strings.HasPrefix(encoded, "b64!") {
		t.Fatalf("expected b64 cursor, got %q", encoded)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, "b64!"))
	if err != nil {
		t.Fatalf("decode cursor transport: %v", err)
	}
	return string(raw)
}
