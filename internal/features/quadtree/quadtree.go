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
func rangeToQuadtreeBinary(targetDStart, targetDEnd int64, maxN int64) string {
	if targetDStart >= targetDEnd {
		return ""
	}

	var buffer []byte
	currentByte := byte(0)
	bitsUsed := 0

	addMask := func(mask byte) {
		if bitsUsed == 0 {
			currentByte = mask << 4
			bitsUsed = 4
		} else {
			currentByte |= mask
			buffer = append(buffer, currentByte)
			currentByte = 0
			bitsUsed = 0
		}
	}

	var processNode func(nodeDStart, nodeDEnd int64, level int)
	processNode = func(nodeDStart, nodeDEnd int64, level int) {
		if nodeDEnd <= targetDStart || nodeDStart >= targetDEnd {
			return
		}

		if nodeDStart >= targetDStart && nodeDEnd <= targetDEnd {
			addMask(0)
			return
		}

		midPoint := (nodeDEnd - nodeDStart) / 4
		childMask := byte(0)

		for i := int64(0); i < 4; i++ {
			childStart := nodeDStart + (i * midPoint)
			childEnd := childStart + midPoint

			if childEnd > targetDStart && childStart < targetDEnd {
				childMask |= 1 << i
			}
		}

		addMask(childMask)

		for i := int64(0); i < 4; i++ {
			if childMask&(1<<i) != 0 {
				childStart := nodeDStart + (i * midPoint)
				childEnd := childStart + midPoint
				processNode(childStart, childEnd, level+1)
			}
		}
	}

	processNode(0, maxN*maxN, 0)

	if bitsUsed > 0 {
		buffer = append(buffer, currentByte)
	}

	return base64.StdEncoding.EncodeToString(buffer)
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

	encoded := rangeToQuadtreeBinary(startD, endD, gridSize)

	return encoded, nil
})

