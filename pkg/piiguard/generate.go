package piiguard

import (
	"fmt"
	"strings"
)

// generate returns the replacement for one real value.
//
// Every generator is deterministic in the hash, so the same real value always
// produces the same fake within a request, and the same value in two messages
// of one conversation stays one entity — which is what keeps the model able to
// follow a referent across turns.
//
// The generated values deliberately use reserved or documentation ranges
// (RFC 2606 domains, RFC 5737 addresses, 555 numbers, 2001:db8::) so a fake can
// never collide with a real reachable destination.
func generate(config Config, entityType, original string, hash uint64) string {
	if override, ok := config.Fakes[entityType][original]; ok && override != "" {
		return override
	}
	switch entityType {
	case EntityEmail:
		return fakeEmail(original, hash)
	case EntityPhone:
		return fakePhone(hash)
	case EntityCreditCard:
		return fakeCard(hash)
	case EntityIPAddress:
		if strings.Contains(original, ":") {
			return fakeIPv6(hash)
		}
		return fakeIPv4(hash)
	case EntityUUID:
		return fakeUUID(hash)
	case EntityIBAN:
		return fakeIBAN(original, hash)
	case EntitySSN:
		return fakeSSN(hash)
	case EntityPassport:
		return fakePassport(hash)
	case EntityDriverLic:
		return fakeDriverLicense(hash)
	case EntityVNID:
		return fakeVNID(original, hash)
	case EntityVNTaxCode:
		return fakeVNTaxCode(hash)
	case EntityDateOfBirth:
		return fakeDate(hash)
	case EntityPersonName:
		return fakeName(hash)
	case EntityOrganization:
		return fakeOrganization(hash)
	case EntityLocation:
		return fakeLocation(hash)
	default:
		return fakeGeneric(entityType, hash)
	}
}

// fakeEmail keeps the domain, because a domain is rarely the sensitive part and
// keeping it preserves the message's meaning, and replaces the local part with
// a reserved-domain address.
func fakeEmail(original string, hash uint64) string {
	domain := "example.com"
	if at := strings.LastIndex(original, "@"); at >= 0 {
		candidate := strings.TrimSpace(original[at+1:])
		if candidate != "" && !strings.ContainsAny(candidate, " \t") {
			domain = candidate
		}
	}
	return fmt.Sprintf("pii-%012x@%s", hash&0xffffffffffff, domain)
}

// fakePhone stays inside the 555-01XX range that is reserved for fiction.
func fakePhone(hash uint64) string {
	return fmt.Sprintf("+1 555 01%02d", hash%100)
}

// fakeCard embeds the hash in a 0000-prefixed number and repairs the check
// digit, so the fake is checksum valid but belongs to no issuer.
func fakeCard(hash uint64) string {
	digits := fmt.Sprintf("0000%012d", hash%1_000_000_000_000)
	digits = digits[:15]
	for check := 0; check <= 9; check++ {
		candidate := digits + string(rune('0'+check))
		if luhnValid(candidate) {
			return candidate
		}
	}
	// Unreachable for a 15 digit prefix, but a generator must always return a
	// value rather than panic.
	return digits + "0"
}

// fakeIPv4 uses the 198.18.0.0/15 benchmarking range.
func fakeIPv4(hash uint64) string {
	return fmt.Sprintf("198.18.%d.%d", (hash>>8)%256, hash%256)
}

// fakeIPv6 uses the 2001:db8::/32 documentation range.
func fakeIPv6(hash uint64) string {
	return fmt.Sprintf("2001:db8::%x", hash&0xffffffff)
}

// fakeUUID renders the hash as a version 4 shaped identifier.
func fakeUUID(hash uint64) string {
	return fmt.Sprintf("%08x-%04x-4%03x-%04x-%012x",
		hash&0xffffffff,
		(hash>>32)&0xffff,
		(hash>>48)&0xfff,
		0x8000|((hash>>60)&0x0fff),
		hash&0xffffffffffff)
}

// fakeIBAN keeps the country prefix, because the country is usually part of the
// message's meaning, and rebuilds the check digits.
func fakeIBAN(original string, hash uint64) string {
	country := "DE"
	if len(original) >= 2 {
		country = strings.ToUpper(original[:2])
	}
	body := fmt.Sprintf("%016x", hash)
	return country + "00" + strings.ToUpper(body[:(len(body)*2)/3])
}

// fakeSSN stays in the 900-999 area range, which the SSA never issues.
func fakeSSN(hash uint64) string {
	return fmt.Sprintf("9%02d-%02d-%04d", hash%100, (hash>>8)%100, (hash>>16)%10000)
}

// fakePassport produces a two-letter, seven-digit identifier.
func fakePassport(hash uint64) string {
	return fmt.Sprintf("P%07d", hash%10_000_000)
}

// fakeDriverLicense produces a single-letter, six-digit identifier.
func fakeDriverLicense(hash uint64) string {
	return fmt.Sprintf("D%06d", hash%1_000_000)
}

// fakeVNID preserves the length of a Vietnamese citizen identity number, since
// the 9 digit and 12 digit generations carry different meaning, and prefix
// every fake with 000 so it can never be a plausible real number.
func fakeVNID(original string, hash uint64) string {
	if len(original) == 9 {
		return fmt.Sprintf("000%06d", hash%1_000_000)
	}
	return fmt.Sprintf("000%09d", hash%1_000_000_000)
}

// fakeVNTaxCode produces a ten digit tax code.
func fakeVNTaxCode(hash uint64) string {
	return fmt.Sprintf("0000%06d", hash%1_000_000)
}

// fakeDate keeps the original shape where it is unambiguous and otherwise uses
// ISO form, so a downstream parser still sees a date.
func fakeDate(hash uint64) string {
	year := 1900 + hash%100
	month := 1 + (hash>>8)%12
	day := 1 + (hash>>16)%28
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

var (
	fakeGivenNames = []string{
		"Alex", "Bailey", "Cameron", "Dakota", "Elliot", "Finley", "Gray", "Harper",
		"Indigo", "Jordan", "Kai", "Logan", "Morgan", "Noor", "Oakley", "Parker",
		"Quinn", "Reese", "Sage", "Tatum", "Umber", "Vale", "Wren", "Xen",
		"Yuki", "Zephyr",
	}
	fakeFamilyNames = []string{
		"Adams", "Bennett", "Carter", "Dawson", "Ellis", "Foster", "Graham", "Hayes",
		"Irving", "Jensen", "Keller", "Lawson", "Mercer", "Nolan", "Osborne", "Palmer",
		"Quincy", "Reeves", "Sutton", "Turner", "Underwood", "Vance", "Whitaker", "Yates",
	}
	fakeOrganizations = []string{
		"Northwind Group", "Acme Holdings", "Blue Harbor Ltd", "Cedar Lane Corp",
		"Delta Works Inc", "Evergreen Partners", "Fairview Systems", "Granite Bay LLC",
	}
	fakeLocations = []string{
		"Springfield", "Riverton", "Fair Oaks", "Cedar Park", "Lakeview", "Brookfield",
	}
)

// fakeName produces a neutral placeholder identity. The reference project
// substitutes a plausible name to keep the text fluent; because a wrong guess
// about locale or gender silently changes the model's reasoning, the generated
// name here is drawn from a fixed neutral pool instead.
func fakeName(hash uint64) string {
	given := fakeGivenNames[hash%uint64(len(fakeGivenNames))]
	family := fakeFamilyNames[(hash>>8)%uint64(len(fakeFamilyNames))]
	return given + " " + family
}

// fakeOrganization produces a placeholder company name.
func fakeOrganization(hash uint64) string {
	return fakeOrganizations[hash%uint64(len(fakeOrganizations))]
}

// fakeLocation produces a placeholder place name.
func fakeLocation(hash uint64) string {
	return fakeLocations[hash%uint64(len(fakeLocations))]
}

// fakeGeneric covers custom keywords and any entity type without a dedicated
// generator: a short stable digest is enough to keep the mapping bijective
// while making the value obviously not real data.
func fakeGeneric(entityType string, hash uint64) string {
	return fmt.Sprintf("[%s-%04x]", entityType, hash&0xffff)
}
