package piiguard

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chunkedReader returns fixed size pieces, which is how an SSE body arrives
// when a fake happens to straddle two reads.
type chunkedReader struct {
	chunks []string
	index  int
}

func (r *chunkedReader) Read(buffer []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}
	chunk := r.chunks[r.index]
	r.index++
	return copy(buffer, chunk), nil
}

func TestUnmaskReaderRestoresAcrossChunkBoundaries(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	masked := engine.MaskText("write to alex@example.com today")
	require.NotContains(t, masked, "alex@example.com")

	// Split the masked text into single byte chunks so every replacement is
	// cut apart.
	chunks := make([]string, 0, len(masked))
	for index := 0; index < len(masked); index++ {
		chunks = append(chunks, masked[index:index+1])
	}
	reader := engine.NewUnmaskReader(&chunkedReader{chunks: chunks})

	restored, err := io.ReadAll(reader)

	require.NoError(t, err)
	assert.Equal(t, "write to alex@example.com today", string(restored))
}

func TestUnmaskReaderPreservesNonMaskedTraffic(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	engine.MaskText("write to alex@example.com")

	payload := `{"choices":[{"delta":{"content":"hello"}}]}`
	reader := engine.NewUnmaskReader(&chunkedReader{chunks: []string{payload[:10], payload[10:]}})

	restored, err := io.ReadAll(reader)

	require.NoError(t, err)
	assert.Equal(t, payload, string(restored))
}

func TestUnmaskReaderReturnsSourceWhenNothingWasMasked(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	source := &chunkedReader{chunks: []string{"plain"}}

	assert.Same(t, io.Reader(source), engine.NewUnmaskReader(source))
}

func TestUnmaskReaderKeepsAFakeAtTheVeryEnd(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	masked := engine.MaskText("mail alex@example.com")

	reader := engine.NewUnmaskReader(strings.NewReader(masked))
	restored, err := io.ReadAll(reader)

	require.NoError(t, err)
	assert.Equal(t, "mail alex@example.com", string(restored))
}
