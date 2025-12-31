package index

import (
	"testing"
	"github.com/go-git/go-git/v5/plumbing"
)

//func TestCountLines(t *testing.T) {
//	tests := []struct {
//		name     string
//		content  string
//		expected int64
//	}{
//		{
//			name:     "Empty content",
//			content:  "",
//			expected: 0,
//		},
//		{
//			name:     "Single line without newline",
//			content:  "hello",
//			expected: 1,
//		},
//		{
//			name:     "Single line with newline",
//			content:  "hello\n",
//			expected: 1,
//		},
//		{
//			name:     "Multiple lines",
//			content:  "line1\nline2\nline3\n",
//			expected: 3,
//		},
//		{
//			name:     "Multiple lines without final newline",
//			content:  "line1\nline2\nline3",
//			expected: 3,
//		},
//		{
//			name:     "Empty lines included",
//			content:  "line1\n\nline3\n",
//			expected: 3,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			reader := strings.NewReader(tt.content)
//			count, err := countLines(reader)
//			if err != nil {
//				t.Fatalf("countLines failed: %v", err)
//			}
//			if count != tt.expected {
//				t.Errorf("Expected %d lines, got %d", tt.expected, count)
//			}
//		})
//	}
//}
//
//func TestComputeIndexBlob(t *testing.T) {
//	fs := memfs.New()
//	repo, err := git.Init(memory.NewStorage(), fs)
//	if err != nil {
//		t.Fatalf("Failed to init git repo: %v", err)
//	}
//
//	worktree, err := repo.Worktree()
//	if err != nil {
//		t.Fatalf("Failed to get worktree: %v", err)
//	}
//
//	tests := []struct {
//		name          string
//		content       string
//		expectedLines int64
//	}{
//		{
//			name:          "Simple file",
//			content:       "Hello, World!\nThis is a test.",
//			expectedLines: 2,
//		},
//		{
//			name:          "Single line",
//			content:       "Single line without newline",
//			expectedLines: 1,
//		},
//		{
//			name:          "Empty file",
//			content:       "",
//			expectedLines: 0,
//		},
//		{
//			name:          "File with many lines",
//			content:       "line1\nline2\nline3\nline4\nline5\n",
//			expectedLines: 5,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			fileName := tt.name + ".txt"
//
//			testFile, err := fs.Create(fileName)
//			if err != nil {
//				t.Fatalf("Failed to create test file: %v", err)
//			}
//
//			_, err = testFile.Write([]byte(tt.content))
//			if err != nil {
//				t.Fatalf("Failed to write test content: %v", err)
//			}
//			testFile.Close()
//
//			_, err = worktree.Add(fileName)
//			if err != nil {
//				t.Fatalf("Failed to add file: %v", err)
//			}
//
//			commitHash, err := worktree.Commit("Test commit", &git.CommitOptions{
//				Author: &object.Signature{
//					Name:  "Test User",
//					Email: "test@example.com",
//					When:  time.Now(),
//				},
//			})
//			if err != nil {
//				t.Fatalf("Failed to create commit: %v", err)
//			}
//
//			commit, err := repo.CommitObject(commitHash)
//			if err != nil {
//				t.Fatalf("Failed to get commit object: %v", err)
//			}
//
//			tree, err := commit.Tree()
//			if err != nil {
//				t.Fatalf("Failed to get tree: %v", err)
//			}
//
//			entry, err := tree.FindEntry(fileName)
//			if err != nil {
//				t.Fatalf("Failed to find file entry: %v", err)
//			}
//
//			index, err := ComputeIndex(context.Background(), repo, entry.Hash)
//			if err != nil {
//				t.Fatalf("ComputeIndex failed: %v", err)
//			}
//
//			if len(index.Entries) != 1 {
//				t.Fatalf("Expected 1 entry, got %d", len(index.Entries))
//			}
//
//			entry1 := index.Entries[0]
//			if entry1.Path != "." {
//				t.Errorf("Expected path '.', got '%s'", entry1.Path)
//			}
//
//			if entry1.LineOffset != 0 {
//				t.Errorf("Expected LineOffset 0, got %d", entry1.LineOffset)
//			}
//
//			if entry1.LineCount != tt.expectedLines {
//				t.Errorf("Expected LineCount %d, got %d", tt.expectedLines, entry1.LineCount)
//			}
//		})
//	}
//}
//
//func TestComputeIndexTree(t *testing.T) {
//	fs := memfs.New()
//	repo, err := git.Init(memory.NewStorage(), fs)
//	if err != nil {
//		t.Fatalf("Failed to init git repo: %v", err)
//	}
//
//	worktree, err := repo.Worktree()
//	if err != nil {
//		t.Fatalf("Failed to get worktree: %v", err)
//	}
//
//	// Create directory structure:
//	// /a.txt (2 lines)
//	// /b.txt (3 lines)
//	// /dir/
//	//   /c.txt (1 line)
//	//   /d.txt (4 lines)
//
//	files := map[string]string{
//		"a.txt":     "line1\nline2\n",        // 2 lines
//		"b.txt":     "line1\nline2\nline3\n", // 3 lines
//		"dir/c.txt": "single line",           // 1 line
//		"dir/d.txt": "1\n2\n3\n4\n",          // 4 lines
//	}
//
//	// Create files
//	for path, content := range files {
//		if strings.Contains(path, "/") {
//			dir := path[:strings.LastIndex(path, "/")]
//			err = fs.MkdirAll(dir, 0755)
//			if err != nil {
//				t.Fatalf("Failed to create dir %s: %v", dir, err)
//			}
//		}
//
//		testFile, err := fs.Create(path)
//		if err != nil {
//			t.Fatalf("Failed to create file %s: %v", path, err)
//		}
//
//		_, err = testFile.Write([]byte(content))
//		if err != nil {
//			t.Fatalf("Failed to write content to %s: %v", path, err)
//		}
//		testFile.Close()
//
//		_, err = worktree.Add(path)
//		if err != nil {
//			t.Fatalf("Failed to add file %s: %v", path, err)
//		}
//	}
//
//	commitHash, err := worktree.Commit("Test commit", &git.CommitOptions{
//		Author: &object.Signature{
//			Name:  "Test User",
//			Email: "test@example.com",
//			When:  time.Now(),
//		},
//	})
//	if err != nil {
//		t.Fatalf("Failed to create commit: %v", err)
//	}
//
//	commit, err := repo.CommitObject(commitHash)
//	if err != nil {
//		t.Fatalf("Failed to get commit object: %v", err)
//	}
//
//	tree, err := commit.Tree()
//	if err != nil {
//		t.Fatalf("Failed to get tree: %v", err)
//	}
//
//	// Test root tree
//	index, err := ComputeIndex(context.Background(), repo, tree.Hash)
//	if err != nil {
//		t.Fatalf("ComputeIndex failed: %v", err)
//	}
//
//	// Expected entries in lexicographic order:
//	// a.txt: LineOffset=0, LineCount=2
//	// b.txt: LineOffset=2, LineCount=3
//	// dir/c.txt: LineOffset=5, LineCount=1
//	// dir/d.txt: LineOffset=6, LineCount=4
//
//	expectedEntries := []IndexEntry{
//		{Path: "a.txt", LineOffset: 0, LineCount: 2},
//		{Path: "b.txt", LineOffset: 2, LineCount: 3},
//		{Path: "dir/c.txt", LineOffset: 5, LineCount: 1},
//		{Path: "dir/d.txt", LineOffset: 6, LineCount: 4},
//	}
//
//	if len(index.Entries) != len(expectedEntries) {
//		t.Fatalf("Expected %d entries, got %d", len(expectedEntries), len(index.Entries))
//	}
//
//	for i, expected := range expectedEntries {
//		actual := index.Entries[i]
//		if actual.Path != expected.Path {
//			t.Errorf("Entry %d: expected path %s, got %s", i, expected.Path, actual.Path)
//		}
//		if actual.LineOffset != expected.LineOffset {
//			t.Errorf("Entry %d (%s): expected LineOffset %d, got %d", i, expected.Path, expected.LineOffset, actual.LineOffset)
//		}
//		if actual.LineCount != expected.LineCount {
//			t.Errorf("Entry %d (%s): expected LineCount %d, got %d", i, expected.Path, expected.LineCount, actual.LineCount)
//		}
//	}
//}
//
//func TestComputeIndexSubtree(t *testing.T) {
//	fs := memfs.New()
//	repo, err := git.Init(memory.NewStorage(), fs)
//	if err != nil {
//		t.Fatalf("Failed to init git repo: %v", err)
//	}
//
//	worktree, err := repo.Worktree()
//	if err != nil {
//		t.Fatalf("Failed to get worktree: %v", err)
//	}
//
//	// Create directory structure:
//	// /subdir/
//	//   /x.txt (2 lines)
//	//   /y.txt (1 line)
//
//	err = fs.MkdirAll("subdir", 0755)
//	if err != nil {
//		t.Fatalf("Failed to create subdir: %v", err)
//	}
//
//	files := map[string]string{
//		"subdir/x.txt": "line1\nline2\n", // 2 lines
//		"subdir/y.txt": "single",         // 1 line
//	}
//
//	for path, content := range files {
//		testFile, err := fs.Create(path)
//		if err != nil {
//			t.Fatalf("Failed to create file %s: %v", path, err)
//		}
//
//		_, err = testFile.Write([]byte(content))
//		if err != nil {
//			t.Fatalf("Failed to write content to %s: %v", path, err)
//		}
//		testFile.Close()
//
//		_, err = worktree.Add(path)
//		if err != nil {
//			t.Fatalf("Failed to add file %s: %v", path, err)
//		}
//	}
//
//	commitHash, err := worktree.Commit("Test commit", &git.CommitOptions{
//		Author: &object.Signature{
//			Name:  "Test User",
//			Email: "test@example.com",
//			When:  time.Now(),
//		},
//	})
//	if err != nil {
//		t.Fatalf("Failed to create commit: %v", err)
//	}
//
//	commit, err := repo.CommitObject(commitHash)
//	if err != nil {
//		t.Fatalf("Failed to get commit object: %v", err)
//	}
//
//	tree, err := commit.Tree()
//	if err != nil {
//		t.Fatalf("Failed to get tree: %v", err)
//	}
//
//	// Get the subdir tree
//	subdirEntry, err := tree.FindEntry("subdir")
//	if err != nil {
//		t.Fatalf("Failed to find subdir entry: %v", err)
//	}
//
//	index, err := ComputeIndex(context.Background(), repo, subdirEntry.Hash)
//	if err != nil {
//		t.Fatalf("ComputeIndex failed: %v", err)
//	}
//
//	// Expected entries for subtree (lexicographic order):
//	// x.txt: LineOffset=0, LineCount=2
//	// y.txt: LineOffset=2, LineCount=1
//
//	expectedEntries := []IndexEntry{
//		{Path: "x.txt", LineOffset: 0, LineCount: 2},
//		{Path: "y.txt", LineOffset: 2, LineCount: 1},
//	}
//
//	if len(index.Entries) != len(expectedEntries) {
//		t.Fatalf("Expected %d entries, got %d", len(expectedEntries), len(index.Entries))
//	}
//
//	for i, expected := range expectedEntries {
//		actual := index.Entries[i]
//		if actual.Path != expected.Path {
//			t.Errorf("Entry %d: expected path %s, got %s", i, expected.Path, actual.Path)
//		}
//		if actual.LineOffset != expected.LineOffset {
//			t.Errorf("Entry %d (%s): expected LineOffset %d, got %d", i, expected.Path, expected.LineOffset, actual.LineOffset)
//		}
//		if actual.LineCount != expected.LineCount {
//			t.Errorf("Entry %d (%s): expected LineCount %d, got %d", i, expected.Path, expected.LineCount, actual.LineCount)
//		}
//	}
//}
//
//func TestComputeIndexEmptyTree(t *testing.T) {
//	fs := memfs.New()
//	repo, err := git.Init(memory.NewStorage(), fs)
//	if err != nil {
//		t.Fatalf("Failed to init git repo: %v", err)
//	}
//
//	worktree, err := repo.Worktree()
//	if err != nil {
//		t.Fatalf("Failed to get worktree: %v", err)
//	}
//
//	// Create a commit with no files
//	commitHash, err := worktree.Commit("Empty commit", &git.CommitOptions{
//		Author: &object.Signature{
//			Name:  "Test User",
//			Email: "test@example.com",
//			When:  time.Now(),
//		},
//		AllowEmptyCommits: true,
//	})
//	if err != nil {
//		t.Fatalf("Failed to create empty commit: %v", err)
//	}
//
//	commit, err := repo.CommitObject(commitHash)
//	if err != nil {
//		t.Fatalf("Failed to get commit object: %v", err)
//	}
//
//	tree, err := commit.Tree()
//	if err != nil {
//		t.Fatalf("Failed to get tree: %v", err)
//	}
//
//	index, err := ComputeIndex(context.Background(), repo, tree.Hash)
//	if err != nil {
//		t.Fatalf("ComputeIndex failed: %v", err)
//	}
//
//	if len(index.Entries) != 0 {
//		t.Errorf("Expected 0 entries for empty tree, got %d", len(index.Entries))
//	}
//}

func TestFindFileByLine(t *testing.T) {
	// Create test index with multiple files
	// File structure:
	// file1.txt: lines 0-2 (3 lines)
	// file2.txt: lines 3-4 (2 lines)
	// dir/file3.txt: lines 5-9 (5 lines)
	// file4.txt: lines 10-10 (1 line)
	
	hash1 := plumbing.NewHash("1111111111111111111111111111111111111111")
	hash2 := plumbing.NewHash("2222222222222222222222222222222222222222")
	hash3 := plumbing.NewHash("3333333333333333333333333333333333333333")
	hash4 := plumbing.NewHash("4444444444444444444444444444444444444444")
	
	index := Index{
		Entries: []IndexEntry{
			{Path: "file1.txt", LineOffset: 0, LineCount: 3, Hash: hash1},
			{Path: "file2.txt", LineOffset: 3, LineCount: 2, Hash: hash2},
			{Path: "dir/file3.txt", LineOffset: 5, LineCount: 5, Hash: hash3},
			{Path: "file4.txt", LineOffset: 10, LineCount: 1, Hash: hash4},
		},
	}
	
	tests := []struct {
		name         string
		lineNumber   int64
		expectedPath string
		shouldFind   bool
	}{
		// Test finding files by line number
		{"First line of first file", 0, "file1.txt", true},
		{"Middle line of first file", 1, "file1.txt", true},
		{"Last line of first file", 2, "file1.txt", true},
		
		{"First line of second file", 3, "file2.txt", true},
		{"Last line of second file", 4, "file2.txt", true},
		
		{"First line of third file", 5, "dir/file3.txt", true},
		{"Middle line of third file", 7, "dir/file3.txt", true},
		{"Last line of third file", 9, "dir/file3.txt", true},
		
		{"Only line of fourth file", 10, "file4.txt", true},
		
		// Test edge cases
		{"Negative line number", -1, "", false},
		{"Line after all files", 11, "", false},
		{"Line in gap between files (impossible with valid index)", 100, "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := index.FindFileByLine(tt.lineNumber)
			
			if tt.shouldFind {
				if result == nil {
					t.Errorf("Expected to find file for line %d, but got nil", tt.lineNumber)
					return
				}
				if result.Path != tt.expectedPath {
					t.Errorf("Expected path %s for line %d, got %s", tt.expectedPath, tt.lineNumber, result.Path)
				}
				// Verify the line is actually within the range
				if tt.lineNumber < result.LineOffset || tt.lineNumber >= result.LineOffset+result.LineCount {
					t.Errorf("Line %d is not within entry range [%d, %d)", tt.lineNumber, result.LineOffset, result.LineOffset+result.LineCount)
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil for line %d, but got entry for path %s", tt.lineNumber, result.Path)
				}
			}
		})
	}
}

func TestFindFileByLineEmptyIndex(t *testing.T) {
	index := Index{Entries: []IndexEntry{}}
	
	result := index.FindFileByLine(0)
	if result != nil {
		t.Errorf("Expected nil for empty index, got %v", result)
	}
	
	result = index.FindFileByLine(5)
	if result != nil {
		t.Errorf("Expected nil for empty index, got %v", result)
	}
}

func TestFindFileByLineSingleEntry(t *testing.T) {
	hash1 := plumbing.NewHash("1111111111111111111111111111111111111111")
	index := Index{
		Entries: []IndexEntry{
			{Path: "single.txt", LineOffset: 0, LineCount: 5, Hash: hash1},
		},
	}
	
	tests := []struct {
		name       string
		lineNumber int64
		shouldFind bool
	}{
		{"Line before file", -1, false},
		{"First line", 0, true},
		{"Middle line", 2, true},
		{"Last line", 4, true},
		{"Line after file", 5, false},
		{"Line way after file", 100, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := index.FindFileByLine(tt.lineNumber)
			
			if tt.shouldFind {
				if result == nil {
					t.Errorf("Expected to find file for line %d, but got nil", tt.lineNumber)
				} else if result.Path != "single.txt" {
					t.Errorf("Expected path 'single.txt', got %s", result.Path)
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil for line %d, but got entry for path %s", tt.lineNumber, result.Path)
				}
			}
		})
	}
}

func TestFindFileByLineWithZeroLineCountEntries(t *testing.T) {
	// Test edge case with zero-line files
	hash1 := plumbing.NewHash("1111111111111111111111111111111111111111")
	hash2 := plumbing.NewHash("2222222222222222222222222222222222222222")
	hash3 := plumbing.NewHash("3333333333333333333333333333333333333333")
	
	index := Index{
		Entries: []IndexEntry{
			{Path: "empty1.txt", LineOffset: 0, LineCount: 0, Hash: hash1},
			{Path: "normal.txt", LineOffset: 0, LineCount: 3, Hash: hash2},
			{Path: "empty2.txt", LineOffset: 3, LineCount: 0, Hash: hash3},
		},
	}
	
	tests := []struct {
		name         string
		lineNumber   int64
		expectedPath string
		shouldFind   bool
	}{
		{"Line 0", 0, "normal.txt", true},
		{"Line 1", 1, "normal.txt", true},
		{"Line 2", 2, "normal.txt", true},
		{"Line 3", 3, "", false}, // After normal.txt, empty2.txt has 0 lines
		{"Line 4", 4, "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := index.FindFileByLine(tt.lineNumber)
			
			if tt.shouldFind {
				if result == nil {
					t.Errorf("Expected to find file for line %d, but got nil", tt.lineNumber)
				} else if result.Path != tt.expectedPath {
					t.Errorf("Expected path %s for line %d, got %s", tt.expectedPath, tt.lineNumber, result.Path)
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil for line %d, but got entry for path %s", tt.lineNumber, result.Path)
				}
			}
		})
	}
}
