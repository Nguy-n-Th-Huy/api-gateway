package setting

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// adminOnlyGroups lists group names that only administrators and root users may
// resolve or use. It is orthogonal to UserUsableGroups, GroupRatio and
// AutoGroups: marking a group here changes none of those settings. An empty
// list (the default, and the result of absent or malformed configuration) means
// nobody is marked, so resolution behaves exactly as before the feature.
//
// The list is guarded by adminOnlyGroupsMutex because the writer is reached from
// the option-update path (model/option.go), which loadOptionsFromDatabase
// re-applies for every option on the SyncOptions timer goroutine as well as on a
// settings save. Readers run on every relay, pricing, group-list and model-list
// request, so an unguarded slice header would be a data race: a torn read makes
// ContainsAdminOnlyGroup iterate a stale pointer with a new length, and
// encoding/json reuses the existing backing array when the capacity suffices, so
// it can rewrite string headers in place under a concurrent reader. Either one
// can transiently report an admin-only group as not admin-only — the exact value
// this setting exists to protect.
var adminOnlyGroups = []string{}
var adminOnlyGroupsMutex sync.RWMutex

// ContainsAdminOnlyGroup reports whether a single group name is marked
// admin-only.
func ContainsAdminOnlyGroup(group string) bool {
	adminOnlyGroupsMutex.RLock()
	defer adminOnlyGroupsMutex.RUnlock()

	for _, adminOnlyGroup := range adminOnlyGroups {
		if adminOnlyGroup == group {
			return true
		}
	}
	return false
}

// UpdateAdminOnlyGroupsByJsonString replaces the admin-only group list from a
// JSON array. The slice is reset before unmarshalling so a malformed payload
// leaves it empty rather than partially populated, mirroring
// UpdateAutoGroupsByJsonString. Absent configuration never reaches this path and
// keeps the empty default.
func UpdateAdminOnlyGroupsByJsonString(jsonString string) error {
	adminOnlyGroupsMutex.Lock()
	defer adminOnlyGroupsMutex.Unlock()

	adminOnlyGroups = make([]string, 0)
	return common.Unmarshal([]byte(jsonString), &adminOnlyGroups)
}

// AdminOnlyGroups2JsonString serializes the current admin-only group list.
func AdminOnlyGroups2JsonString() string {
	adminOnlyGroupsMutex.RLock()
	defer adminOnlyGroupsMutex.RUnlock()

	jsonBytes, err := common.Marshal(adminOnlyGroups)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// GetAdminOnlyGroups returns a copy of the marked group names, so a caller can
// never alias — and therefore mutate or retain — the guarded slice.
func GetAdminOnlyGroups() []string {
	adminOnlyGroupsMutex.RLock()
	defer adminOnlyGroupsMutex.RUnlock()

	groups := make([]string, len(adminOnlyGroups))
	copy(groups, adminOnlyGroups)
	return groups
}
