package piiguard

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// detector finds candidate spans in a single text value.
type detector interface {
	// entityType is the entity every span of this detector resolves to.
	entityType() string
	// match returns the candidate spans of this text.
	match(text string) []span
}

// span is one candidate occurrence of an entity type.
type span struct {
	start int
	end   int
	kind  string
	// requiredContext, when non-empty, forces the span to carry a corroborating
	// keyword in its own neighbourhood. It is what keeps a bare 9 digit number
	// from being reported as a national ID.
	requiredContext []string
	// priority breaks ties between candidates that cover the same span.
	priority int
}

// Detector priorities. A higher value wins when two candidates cover the same
// bytes, which is why the unambiguous identifiers outrank the broad ones.
const (
	priorityLow    = 10
	priorityNormal = 20
	priorityHigh   = 30
)

// Pattern definitions.
var (
	// Addresses and identifiers.
	emailPattern  = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]{0,61}[A-Za-z0-9])?)+`)
	urlPattern    = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>"'\\]{2,}`)
	uuidPattern   = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	ibanPattern   = regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{10,30}\b`)
	ipv4Pattern   = regexp.MustCompile(`\b(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\b`)
	// An IPv6 address needs a compressed group, a leading double colon, or the
	// full eight groups. Without that requirement a clock reading such as
	// 12:30:45 would look like an address.
	ipv6Patterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){1,7}:(?:[0-9a-f]{1,4}(?::[0-9a-f]{1,4}){0,6})?\b`),
		regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){7}[0-9a-f]{1,4}\b`),
	}
	ssnPattern    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	creditPattern = regexp.MustCompile(`\b\d(?:[\s\-]?\d){12,18}\b`)
	vnTaxPattern  = regexp.MustCompile(`\b\d{10}(?:-\d{3})?\b`)

	// Phone numbers. The international form is unambiguous; the Vietnamese
	// local form is gated on a context word in its neighbourhood.
	phoneIntlPattern = regexp.MustCompile(`\+\d{1,3}[\s.\-]?(?:\(\d{1,4}\)[\s.\-]?)?\d(?:[\s.\-]?\d){5,12}\b`)
	phoneVNPattern   = regexp.MustCompile(`\b0\d{9,10}\b`)

	// Context-gated identifiers.
	vnIDPattern       = regexp.MustCompile(`\b\d{9}\b|\b\d{12}\b`)
	passportPattern   = regexp.MustCompile(`\b[A-Z]{1,2}\d{6,9}\b`)
	driverLicPattern  = regexp.MustCompile(`\b[A-Z]{1,2}\d{5,7}\b`)
	dobPattern        = regexp.MustCompile(`\b\d{1,2}[/\-.]\d{1,2}[/\-.]\d{2,4}\b|\b\d{4}-\d{2}-\d{2}\b`)
	dateSpelledFormat = regexp.MustCompile(`\b(?:0?[1-9]|[12]\d|3[01])\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4}\b`)
)

// Context keyword tables. Keywords are matched case-insensitively on a window
// of text around the candidate span.
var (
	vnIDContext       = []string{"cccd", "cmnd", "căn cước", "can cuoc", "chứng minh", "chung minh", "citizen id", "national id", "identity card"}
	passportContext   = []string{"passport", "hộ chiếu", "ho chieu", "travel document"}
	driverLicContext  = []string{"driver", "driving licence", "driving license", "lái xe", "bang lai", "bằng lái", "gplx"}
	dateOfBirthCtx    = []string{"date of birth", "birth date", "birthday", "dob", "born", "ngày sinh", "ngay sinh", "sinh ngày", "sinh ngay"}
	vnTaxCodeContext  = []string{"tax", "mst", "mã số thuế", "ma so thue", "vat"}
	phoneVNContext    = []string{"phone", "tel", "mobile", "điện thoại", "dien thoai", "sđt", "sdt", "liên hệ", "lien he", "zalo", "whatsapp"}
	organizationCtx   = []string{"company", "corp", "corporation", "inc", "ltd", "llc", "gmbh", "hospital", "bệnh viện", "benh vien", "university", "đại học", "cong ty", "công ty", "employer", "bank", "ngân hàng", "ngan hang"}
	locationCtx       = []string{"address", "address:", "địa chỉ", "dia chi", "city", "thành phố", "thanh pho", "street", "đường", "duong", "ward", "phường", "phuong", "district", "quận", "quan", "province", "tỉnh", "tinh"}
	nameSalutations   = []string{"mr", "mrs", "ms", "miss", "dr", "prof", "ông", "ong", "bà", "ba", "anh", "chị", "chi", "em", "cô", "co", "chú", "chu", "bác", "bac", "ngài", "ngai"}
	nameStopwordBlack = map[string]struct{}{
		"anh": {}, "em": {}, "chi": {}, "ba": {}, "co": {}, "chu": {}, "bac": {}, "ong": {},
		"the": {}, "and": {}, "for": {}, "you": {}, "your": {}, "with": {}, "this": {}, "that": {},
	}
)

// contextWindow is how many bytes around a candidate are searched for its
// corroborating keyword.
const contextWindow = 48

// defaultDetectors returns the pattern layer. The order sets tie-break
// priority, so the unambiguous identifiers come first.
func defaultDetectors() []detector {
	return []detector{
		&contextDetector{kind: EntityEmail, re: emailPattern, priority: priorityHigh},
		&contextDetector{kind: EntityUUID, re: uuidPattern, priority: priorityHigh},
		&contextDetector{kind: EntityURL, re: urlPattern, priority: priorityHigh},
		&contextDetector{kind: EntityIBAN, re: ibanPattern, priority: priorityHigh, requiredContext: []string{"iban", "account", "tài khoản", "tai khoan", "bank"}},
		&creditCardDetector{},
		&multiPatternDetector{kind: EntityIPAddress, patterns: ipv6Patterns, priority: priorityNormal},
		&contextDetector{kind: EntityIPAddress, re: ipv4Pattern, priority: priorityNormal},
		&contextDetector{kind: EntitySSN, re: ssnPattern, priority: priorityNormal},
		&contextDetector{kind: EntityPhone, re: phoneIntlPattern, priority: priorityHigh},
		&contextDetector{kind: EntityPhone, re: phoneVNPattern, priority: priorityLow, requiredContext: phoneVNContext},
		&contextDetector{kind: EntityVNID, re: vnIDPattern, priority: priorityNormal, requiredContext: vnIDContext},
		&contextDetector{kind: EntityPassport, re: passportPattern, priority: priorityNormal, requiredContext: passportContext},
		&contextDetector{kind: EntityDriverLic, re: driverLicPattern, priority: priorityNormal, requiredContext: driverLicContext},
		&contextDetector{kind: EntityVNTaxCode, re: vnTaxPattern, priority: priorityNormal, requiredContext: vnTaxCodeContext},
		&contextDetector{kind: EntityDateOfBirth, re: dobPattern, priority: priorityNormal, requiredContext: dateOfBirthCtx},
		&contextDetector{kind: EntityDateOfBirth, re: dateSpelledFormat, priority: priorityNormal, requiredContext: dateOfBirthCtx},
		&nameDetector{kind: EntityPersonName},
		&keywordListDetector{kind: EntityOrganization, keywords: organizationCtx, priority: priorityLow},
		&keywordListDetector{kind: EntityLocation, keywords: locationCtx, priority: priorityLow},
	}
}

// defaultEntityEnabled reports whether an entity type participates when the
// operator has expressed no preference. Broad patterns stay off so the shipped
// behaviour is high precision.
func defaultEntityEnabled(entityType string) bool {
	switch entityType {
	case EntityEmail, EntityPhone, EntityCreditCard, EntityIPAddress, EntityUUID, EntityPassport,
		EntityDriverLic, EntityVNID, EntityIBAN, EntityDateOfBirth, EntityCustom:
		return true
	default:
		// URL, US_SSN, VN_TAX_CODE, PERSON, ORGANIZATION and LOCATION are
		// opt-in: each of them needs either a context word or a lexicon, and a
		// wrong guess damages a legitimate message.
		return false
	}
}

// contextDetector reports every match of one pattern as one entity type.
type contextDetector struct {
	kind            string
	re              *regexp.Regexp
	priority        int
	requiredContext []string
}

func (d *contextDetector) entityType() string {
	return d.kind
}

func (d *contextDetector) match(text string) []span {
	matches := d.re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}
	priority := d.priority
	if priority == 0 {
		priority = priorityNormal
	}
	spans := make([]span, 0, len(matches))
	for _, match := range matches {
		spans = append(spans, span{
			start:           match[0],
			end:             match[1],
			kind:            d.kind,
			requiredContext: d.requiredContext,
			priority:        priority,
		})
	}
	return spans
}

// multiPatternDetector reports every match of any of its patterns as one entity
// type. It exists for an entity whose shape has more than one valid form, such
// as an IPv6 address written with or without a compressed group.
type multiPatternDetector struct {
	kind     string
	patterns []*regexp.Regexp
	priority int
}

func (d *multiPatternDetector) entityType() string {
	return d.kind
}

func (d *multiPatternDetector) match(text string) []span {
	priority := d.priority
	if priority == 0 {
		priority = priorityNormal
	}
	var spans []span
	for _, pattern := range d.patterns {
		for _, match := range pattern.FindAllStringIndex(text, -1) {
			spans = append(spans, span{
				start:    match[0],
				end:      match[1],
				kind:     d.kind,
				priority: priority,
			})
		}
	}
	return spans
}

// creditCardDetector keeps only digit runs that satisfy the Luhn checksum, so
// order numbers and timestamps are not reported as cards.
type creditCardDetector struct{}

var creditCardSeparators = strings.NewReplacer(" ", "", "-", "", "\t", "")

func (d *creditCardDetector) entityType() string {
	return EntityCreditCard
}

func (d *creditCardDetector) match(text string) []span {
	matches := creditPattern.FindAllStringIndex(text, -1)
	var spans []span
	for _, match := range matches {
		raw := text[match[0]:match[1]]
		digits := creditCardSeparators.Replace(raw)
		if !luhnValid(digits) {
			continue
		}
		spans = append(spans, span{
			start:    match[0],
			end:      match[1],
			kind:     EntityCreditCard,
			priority: priorityHigh,
		})
	}
	return spans
}

// nameDetector reports person names. Splitting a name needs a model, so this is
// a lexicon plus salutation heuristic: it only fires on a capitalised token run
// next to a salutation, or on a run whose tokens are known Vietnamese given
// names. It stays off unless the operator enables EntityPersonName.
type nameDetector struct {
	kind string
}

func (d *nameDetector) entityType() string {
	return d.kind
}

func (d *nameDetector) match(text string) []span {
	var spans []span
	for _, token := range tokenRuns(text) {
		// A name is the leading run of capitalised words: "Nguyen Van An
		// arrived" starts with the name and ends with the verb.
		nameStart, nameEnd, words, ok := leadingCapitalisedWords(text, token)
		if !ok {
			continue
		}
		if !hasSalutationBefore(text, nameStart) && countKnownGivenNames(words) == 0 {
			continue
		}
		spans = append(spans, span{
			start:    nameStart,
			end:      nameEnd,
			kind:     d.kind,
			priority: priorityLow,
		})
	}
	return spans
}

const maxNameWords = 5

// leadingCapitalisedWords trims a token run to its leading capitalised words
// and reports whether the result can be a name.
func leadingCapitalisedWords(text string, token tokenRun) (int, int, []string, bool) {
	words := strings.Fields(token.value)
	if len(words) == 0 {
		return 0, 0, nil, false
	}
	kept := make([]string, 0, len(words))
	end := token.start
	cursor := token.start
	for _, word := range words {
		first, _ := utf8.DecodeRuneInString(word)
		if !unicode.IsUpper(first) {
			break
		}
		trimmed := strings.Trim(word, ".-'")
		if _, blocked := nameStopwordBlack[strings.ToLower(trimmed)]; blocked {
			break
		}
		if len(trimmed) < minNameWordLength {
			break
		}
		kept = append(kept, word)
		end = cursor + len(word)
		cursor = end + 1
	}
	if len(kept) == 0 || len(kept) > maxNameWords {
		return 0, 0, nil, false
	}
	return token.start, end, kept, true
}

// minNameWordLength keeps single letters and initials out of a detected name.
const minNameWordLength = 2

// keywordListDetector reports a capitalised token run that follows one of the
// given context keywords, for example the value after "company:".
type keywordListDetector struct {
	kind     string
	keywords []string
	priority int
}

func (d *keywordListDetector) entityType() string {
	return d.kind
}

func (d *keywordListDetector) match(text string) []span {
	lower := strings.ToLower(text)
	var spans []span
	for _, keyword := range d.keywords {
		for offset := 0; ; {
			found := strings.Index(lower[offset:], keyword)
			if found < 0 {
				break
			}
			absolute := offset + found
			offset = absolute + len(keyword)
			valueStart := skipFiller(lower, offset)
			token, ok := nextValueRun(text, valueStart)
			if !ok {
				continue
			}
			spans = append(spans, span{
				start:    token.start,
				end:      token.end,
				kind:     d.kind,
				priority: d.priority,
			})
			offset = token.end
		}
	}
	return spans
}
