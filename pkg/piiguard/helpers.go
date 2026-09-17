package piiguard

import (
	"hash/fnv"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// luhnValid reports whether a digit string satisfies the Luhn checksum. It is
// what separates a payment card from any other long number.
func luhnValid(digits string) bool {
	if len(digits) < 12 || len(digits) > 19 {
		return false
	}
	sum := 0
	parity := len(digits) % 2
	for index := 0; index < len(digits); index++ {
		character := digits[index]
		if character < '0' || character > '9' {
			return false
		}
		digit := int(character - '0')
		if index%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}

// tokenRun is a run of word tokens with its byte extent in the source text.
type tokenRun struct {
	start int
	end   int
	value string
}

// tokenRuns splits text into runs of consecutive word tokens that are single
// spaced, which is the shape a person or organisation name takes.
func tokenRuns(text string) []tokenRun {
	var runs []tokenRun
	index := 0
	for index < len(text) {
		start, ok := nextWordStart(text, index)
		if !ok {
			break
		}
		runEnd := start
		cursor := start
		for {
			// cursor always sits on a word start, so the next word begins after
			// the current one and the scan can never stall.
			end, ok := wordEnd(text, cursor)
			if !ok {
				break
			}
			runEnd = end
			following, ok := nextWordStart(text, end)
			if !ok || !isSingleSeparator(text[end:following]) {
				break
			}
			cursor = following
		}
		runs = append(runs, tokenRun{
			start: start,
			end:   runEnd,
			value: text[start:runEnd],
		})
		index = runEnd
	}
	return runs
}

// nextWordStart returns the start of the next word token at or after index.
func nextWordStart(text string, index int) (int, bool) {
	cursor := index
	for cursor < len(text) {
		r, size := utf8.DecodeRuneInString(text[cursor:])
		if isWordRune(r) {
			return cursor, true
		}
		cursor += size
	}
	return 0, false
}

// wordEnd returns the exclusive end of the word token starting at index.
// A period is part of the token only when it links single-letter initials
// ("J. Smith"), so "Mr." stays a salutation instead of swallowing the name
// that follows it.
func wordEnd(text string, index int) (int, bool) {
	cursor := index
	for cursor < len(text) {
		r, size := utf8.DecodeRuneInString(text[cursor:])
		if r == '.' {
			if !isInitialPeriod(text, cursor) {
				break
			}
			cursor += size
			continue
		}
		if !isWordRune(r) {
			break
		}
		cursor += size
	}
	return cursor, cursor > index
}

// isInitialPeriod reports whether the period at index links two initials, for
// example "J. Smith".
func isInitialPeriod(text string, index int) bool {
	if index+1 >= len(text) || text[index+1] != ' ' {
		return false
	}
	before, _ := utf8.DecodeLastRuneInString(text[:index])
	after, _ := utf8.DecodeRuneInString(text[index+2:])
	return unicode.IsUpper(before) && unicode.IsUpper(after)
}

// isWordRune reports whether a rune can be part of a name token. A period is
// not one: wordEnd absorbs it only when it links initials.
func isWordRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	return r == '\'' || r == '-'
}

// isSingleSeparator reports whether the gap between two word tokens is a single
// space, which is what makes two tokens one name instead of two sentences.
func isSingleSeparator(gap string) bool {
	return gap == " "
}

// hasSalutationBefore reports whether a salutation immediately precedes the
// token run, ignoring a single separating space.
func hasSalutationBefore(text string, start int) bool {
	if start == 0 || text[start-1] != ' ' {
		return false
	}
	prefix := strings.ToLower(text[:start-1])
	for _, salutation := range nameSalutations {
		if strings.HasSuffix(prefix, salutation) {
			return true
		}
	}
	return false
}

// countKnownGivenNames counts how many words of the run are members of the
// Vietnamese given-name lexicon.
func countKnownGivenNames(words []string) int {
	count := 0
	for _, word := range words {
		if _, ok := vietnameseGivenNames[strings.ToLower(word)]; ok {
			count++
		}
	}
	return count
}

// vietnameseGivenNames is a deliberately small lexicon of common given names.
// It is only consulted when EntityPersonName is enabled.
var vietnameseGivenNames = map[string]struct{}{
	"an": {}, "anh": {}, "bình": {}, "binh": {}, "chi": {}, "châu": {}, "chau": {},
	"dung": {}, "dũng": {}, "dương": {}, "duong": {}, "giang": {}, "hà": {}, "ha": {},
	"hải": {}, "hai": {}, "hiếu": {}, "hieu": {}, "hoa": {}, "hùng": {}, "hung": {},
	"huy": {}, "hương": {}, "huong": {}, "khánh": {}, "khanh": {}, "lan": {}, "linh": {},
	"long": {}, "mai": {}, "minh": {}, "nam": {}, "nga": {}, "ngọc": {}, "ngoc": {},
	"nguyên": {}, "nguyen": {}, "nhung": {}, "oanh": {}, "phong": {}, "phúc": {}, "phuc": {},
	"phương": {}, "phuong": {}, "quân": {}, "quan": {}, "quang": {}, "quỳnh": {}, "quynh": {},
	"sơn": {}, "son": {}, "tâm": {}, "tam": {}, "thảo": {}, "thao": {}, "thắng": {}, "thang": {},
	"thu": {}, "thủy": {}, "thuy": {}, "trang": {}, "trinh": {}, "trung": {}, "tuấn": {}, "tuan": {},
	"tú": {}, "tu": {}, "vy": {}, "yến": {}, "yen": {}, "hằng": {}, "hang": {},
}

// skipFiller advances past the punctuation and spaces that separate a context
// keyword from the value that follows it.
func skipFiller(text string, index int) int {
	cursor := index
	for cursor < len(text) {
		r, size := utf8.DecodeRuneInString(text[cursor:])
		switch r {
		case ' ', '\t', ':', '-', '–', '—', ',', '(', '"', '\'', '*':
			cursor += size
		default:
			return cursor
		}
	}
	return cursor
}

// nextValueRun reads the capitalised token run that starts at index, which is
// the value a context keyword introduces.
func nextValueRun(text string, index int) (tokenRun, bool) {
	if index >= len(text) {
		return tokenRun{}, false
	}
	start, ok := nextWordStart(text, index)
	if !ok || start != index {
		return tokenRun{}, false
	}
	end, ok := wordEnd(text, start)
	if !ok {
		return tokenRun{}, false
	}
	value := text[start:end]
	first, _ := utf8.DecodeRuneInString(value)
	if !unicode.IsUpper(first) {
		return tokenRun{}, false
	}
	// Extend across the following capitalised words so a multi-word value such
	// as "Bradtke Medical" stays one detection.
	for {
		next, ok := nextWordStart(text, end)
		if !ok || !isSingleSeparator(text[end:next]) {
			break
		}
		nextEnd, ok := wordEnd(text, next)
		if !ok {
			break
		}
		nextValue := text[next:nextEnd]
		first, _ := utf8.DecodeRuneInString(nextValue)
		if !unicode.IsUpper(first) {
			break
		}
		end = nextEnd
	}
	return tokenRun{start: start, end: end, value: text[start:end]}, true
}

// hasContextKeyword reports whether any keyword appears in the window of text
// around the span.
func hasContextKeyword(text string, index int, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	start := index - contextWindow
	if start < 0 {
		start = 0
	}
	lower := strings.ToLower(text[start:index])
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// indexTemplate renders the placeholder pattern for one entity occurrence.
func indexTemplate(template, entityType string, index int) string {
	rendered := strings.ReplaceAll(template, "{{type}}", entityType)
	return strings.ReplaceAll(rendered, "{{index}}", strconv.Itoa(index))
}

// hashValue derives a stable 64 bit hash from the request seed and a value.
func hashValue(seed, entityType, value string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(seed))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(entityType))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(value))
	return hasher.Sum64()
}
