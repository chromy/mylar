package quadtree

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// TestRangeToQuadtreeBinary_EmptyRange tests that empty ranges return nil
func TestRangeToQuadtreeBinary_EmptyRange(t *testing.T) {
	tests := []struct {
		name         string
		targetDStart int64
		targetDEnd   int64
		maxN         int64
	}{
		{"equal start and end", 2, 2, 2},
		{"last number is not part of the range", 4, 5, 2},
		{"starts outside the range", 5, 6, 2},
		{"start after end", 10, 5, 2},
		{"end before range", -2, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rangeToQuadtreeBinary(tt.targetDStart, tt.targetDEnd, tt.maxN)
			if result != nil {
				t.Errorf("expected nil, got %v (base64: %s)", result, base64.StdEncoding.EncodeToString(result))
			}
		})
	}
}

// TestRangeToQuadtreeBinary_4x4_SingleQuadrant tests each quadrant of a 4x4 grid
func TestRangeToQuadtreeBinary_Examples(t *testing.T) {
	tests := []struct {
		name     string
		start    int64
		end      int64
		n        int64
		expected []byte
	}{
		{"all covered", 0, 16, 4, []byte{0x00}},
		{"NW quadrant color", 0, 4, 4, []byte{0x01}},
		{"SW quadrant color", 4, 8, 4, []byte{0x02}},
		{"SE quadrant color", 8, 12, 4, []byte{0x04}},
		{"NE quadrant color", 12, 16, 4, []byte{0x08}},
		{"first cell", 0, 1, 4, []byte{0x11}},
		{"second cell", 1, 2, 4, []byte{0x81}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := rangeToQuadtreeBinary(tt.start, tt.end, 4)

			if !bytes.Equal(tt.expected, actual) {
				t.Errorf("mismatch [%d..%d) on %dx%d=%d was 0x%x expected 0x%x", tt.start, tt.end, tt.n, tt.n, tt.n*tt.n, actual, tt.expected)
			}
		})
	}
}
