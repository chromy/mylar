package quadtree

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/chromy/mylar/internal/core"
	"github.com/chromy/mylar/internal/features/index"
	"github.com/chromy/mylar/internal/features/repo"
	"github.com/go-git/go-git/v5/plumbing"
)

// rangeToQuadtreeBinary directly builds a breadth-first binary encoded quadtree
// for the given Hilbert range using 4-bit child masks
func rangeToQuadtreeBinary(targetDStart, targetDEnd int64, maxN int64) []byte {
	total := maxN * maxN
	if targetDStart >= targetDEnd {
		return nil
	}
	if targetDStart >= total {
		return nil
	}
	if targetDEnd <= 0 {
		return nil
	}

	var buffer []byte
	currentByte := byte(0)
	bitsUsed := 0

	addMask := func(mask byte) {
		if bitsUsed == 0 {
			currentByte = mask
			bitsUsed = 4
		} else {
			currentByte |= mask << 4
			buffer = append(buffer, currentByte)
			currentByte = 0
			bitsUsed = 0
		}
	}

	type Node struct {
		start, end int64
		phase      string
	}

	queue := []Node{{start: 0, end: total, phase: "a"}}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.start >= targetDStart && node.end <= targetDEnd {
			addMask(0)
			continue
		}

		quarter := (node.end - node.start) / 4

		segments := []struct {
			start int64
			end   int64
		}{
			{node.start + quarter*0, node.start + quarter*1},
			{node.start + quarter*1, node.start + quarter*2},
			{node.start + quarter*2, node.start + quarter*3},
			{node.start + quarter*3, node.start + quarter*4},
		}

		nodes := make([]Node, 0, 4)

		switch node.phase {
		case "a":
			nodes = append(nodes, Node{segments[0].start, segments[0].end, "d"})
			nodes = append(nodes, Node{segments[1].start, segments[1].end, "a"})
			nodes = append(nodes, Node{segments[2].start, segments[2].end, "a"})
			nodes = append(nodes, Node{segments[3].start, segments[3].end, "b"})
		case "b":
			nodes = append(nodes, Node{segments[2].start, segments[2].end, "b"})
			nodes = append(nodes, Node{segments[1].start, segments[1].end, "b"})
			nodes = append(nodes, Node{segments[0].start, segments[0].end, "c"})
			nodes = append(nodes, Node{segments[3].start, segments[3].end, "a"})
		case "c":
			nodes = append(nodes, Node{segments[2].start, segments[2].end, "c"})
			nodes = append(nodes, Node{segments[3].start, segments[3].end, "d"})
			nodes = append(nodes, Node{segments[0].start, segments[0].end, "b"})
			nodes = append(nodes, Node{segments[1].start, segments[1].end, "c"})
		case "d":
			nodes = append(nodes, Node{segments[0].start, segments[0].end, "a"})
			nodes = append(nodes, Node{segments[3].start, segments[3].end, "c"})
			nodes = append(nodes, Node{segments[2].start, segments[2].end, "d"})
			nodes = append(nodes, Node{segments[1].start, segments[1].end, "d"})
		}

		childMask := byte(0)

		for i, node := range nodes {
			if node.start < targetDEnd && targetDStart < node.end {
				childMask |= 1 << i

				if node.end-node.start > 1 {
					queue = append(queue, node)
				}
			}
		}

		addMask(childMask)
	}

	if bitsUsed > 0 {
		buffer = append(buffer, currentByte)
	}

	return buffer
}

var GetFileQuadtree = core.RegisterCommitComputation("fileQuadtree", func(ctx context.Context, repoId string, commit plumbing.Hash, hash plumbing.Hash) (string, error) {
	tree, err := repo.CommitToTree(ctx, repoId, commit)
	if err != nil {
		return "", fmt.Errorf("failed to get tree from commit: %w", err)
	}

	idx, err := index.GetIndex(ctx, repoId, tree)
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}

	var targetEntry *index.IndexEntry
	for i := range idx.Entries {
		if idx.Entries[i].Hash == hash {
			targetEntry = &idx.Entries[i]
			break
		}
	}

	if targetEntry == nil {
		return "", fmt.Errorf("file with hash %s not found in index", hash)
	}

	layout := idx.ToTileLayout()
	gridSize := layout.GridSideLength()

	startD := targetEntry.LineOffset
	endD := targetEntry.LineOffset + targetEntry.LineCount

	buffer := rangeToQuadtreeBinary(startD, endD, gridSize)
	encoded := base64.StdEncoding.EncodeToString(buffer)

	return encoded, nil
})
