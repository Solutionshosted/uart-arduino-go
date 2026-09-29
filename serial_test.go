package arduino

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type chunkReader struct {
	data string
	step int
}

// Read()
// Simulates serial packets split across arbitrary read boundaries, including a final read with EOF.
func (r *chunkReader) Read(buffer []byte) (int, error) {
	n := len(r.data)
	if n > r.step {
		n = r.step
	}
	if n > len(buffer) {
		n = len(buffer)
	}
	copy(buffer, r.data[:n])
	r.data = r.data[n:]
	if r.data == "" {
		return n, io.EOF
	}
	return n, nil
}

// TestParseFrame()
// Verifies field normalization and framing without imposing game-specific rules.
func TestParseFrame(t *testing.T) {
	f, ok := ParseFrame("<EVALUATION|AMPEL=green|SOLVED=1>")
	if !ok || f.Type != "EVALUATION" || f.Fields["ampel"] != "green" || f.Fields["solved"] != "1" {
		t.Fatalf("incorrect frame: %+v", f)
	}
	for _, input := range []string{"", "noise", "<>", "<|X=1>", "<STATUS", "STATUS>"} {
		if _, ok := ParseFrame(input); ok {
			t.Errorf("accepted malformed frame %q", input)
		}
	}
}

// TestReadSerialChunks()
// Reassembles CRLF frames from small reads and keeps the final complete frame when EOF accompanies data.
func TestReadSerialChunks(t *testing.T) {
	r := &chunkReader{data: "noise\n<STATUS|EVENT=RESET>\r\n<EVALUATION|AMPEL=red|SOLVED=0>\n", step: 3}
	var frames []Frame
	err := readSerial(context.Background(), r, func(f Frame) { frames = append(frames, f) })
	if !errors.Is(err, io.EOF) || len(frames) != 2 || frames[0].Fields["event"] != "RESET" || frames[1].Fields["ampel"] != "red" {
		t.Fatalf("got %d frames, error %v", len(frames), err)
	}
}

// TestReadSerialDiscardsOversizedLines()
// Discards the entire oversized line so a valid-looking suffix cannot become an unintended event.
func TestReadSerialDiscardsOversizedLines(t *testing.T) {
	input := strings.Repeat("x", 4097) + "<STATUS|EVENT=RESET>\n<STATUS|EVENT=GAME_START>\n"
	var frames []Frame
	_ = readSerial(context.Background(), strings.NewReader(input), func(f Frame) { frames = append(frames, f) })
	if len(frames) != 1 || frames[0].Fields["event"] != "GAME_START" {
		t.Fatalf("unexpected frames: %+v", frames)
	}
}

// TestReadSerialCancellation()
// Stops before reading or delivering any messages when the context has already been cancelled.
func TestReadSerialCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := readSerial(ctx, strings.NewReader("<STATUS|EVENT=RESET>\n"), func(Frame) { t.Fatal("unexpected callback") })
	if err != nil {
		t.Fatal(err)
	}
}
