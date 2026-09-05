package setting

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminOnlyGroupsConcurrentReadersAndTimerWriter reproduces the production
// shape of the admin-only setting: readers on every relay, pricing, group-list
// and model-list request, and a writer that runs with no operator action at all
// because model/option.go re-applies every stored option on the SyncOptions
// timer goroutine. It exists to be caught by `go test -race`; without the
// RWMutex in admin_only_group.go the race detector reports the unsynchronised
// slice header and the in-place JSON rewrite of the backing array.
func TestAdminOnlyGroupsConcurrentReadersAndTimerWriter(t *testing.T) {
	original := AdminOnlyGroups2JsonString()
	t.Cleanup(func() {
		require.NoError(t, UpdateAdminOnlyGroupsByJsonString(original))
	})
	require.NoError(t, UpdateAdminOnlyGroupsByJsonString(`[]`))

	const (
		readerCount  = 8
		readerRounds = 250
		writerRounds = 250
	)
	// The writer alternates between two whole published states, which is what
	// re-applying the stored option does on every sync tick.
	states := []string{`["secret","vip"]`, `[]`}

	release := make(chan struct{})
	var readers sync.WaitGroup
	for i := 0; i < readerCount; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-release
			for round := 0; round < readerRounds; round++ {
				// Every snapshot must be one of the two published states. A torn
				// slice header would surface a half-written length or a stale
				// backing array whose string headers no longer match.
				snapshot := GetAdminOnlyGroups()
				if !assert.Contains(t, []int{0, 2}, len(snapshot), "snapshot must be a whole published state") {
					return
				}
				for _, name := range snapshot {
					assert.Contains(t, []string{"secret", "vip"}, name)
				}
				// The membership check is the security-critical read: a group nobody
				// marked must never be reported as admin-only.
				assert.False(t, ContainsAdminOnlyGroup("default"))
			}
		}()
	}

	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		<-release
		for round := 0; round < writerRounds; round++ {
			assert.NoError(t, UpdateAdminOnlyGroupsByJsonString(states[round%len(states)]))
		}
		// Publish a known final state so the post-join assertions are deterministic.
		assert.NoError(t, UpdateAdminOnlyGroupsByJsonString(states[0]))
	}()

	close(release)
	writer.Wait()
	readers.Wait()

	require.Equal(t, []string{"secret", "vip"}, GetAdminOnlyGroups())
	assert.True(t, ContainsAdminOnlyGroup("secret"))
	assert.True(t, ContainsAdminOnlyGroup("vip"))
	assert.False(t, ContainsAdminOnlyGroup("default"))
}

// TestGetAdminOnlyGroupsReturnsCopy pins the other half of the guard: the getter
// must not hand out the protected slice, otherwise a caller could rewrite it
// while readers are inside ContainsAdminOnlyGroup.
func TestGetAdminOnlyGroupsReturnsCopy(t *testing.T) {
	original := AdminOnlyGroups2JsonString()
	t.Cleanup(func() {
		require.NoError(t, UpdateAdminOnlyGroupsByJsonString(original))
	})

	require.NoError(t, UpdateAdminOnlyGroupsByJsonString(`["secret","vip"]`))
	groups := GetAdminOnlyGroups()
	require.Equal(t, []string{"secret", "vip"}, groups)

	groups[0] = "tampered"

	assert.Equal(t, []string{"secret", "vip"}, GetAdminOnlyGroups(), "callers must not alias the guarded slice")
	assert.True(t, ContainsAdminOnlyGroup("secret"))
	assert.False(t, ContainsAdminOnlyGroup("tampered"))
}
