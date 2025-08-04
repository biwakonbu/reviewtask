package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsInitialized tests the initialization check
func TestIsInitialized(t *testing.T) {
	scenarios := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name: "未初期化のリポジトリ",
			setup: func(t *testing.T, dir string) {
				// 何もしない - 空のディレクトリ
			},
			expected: false,
		},
		{
			name: "初期化済みのリポジトリ（config.jsonあり）",
			setup: func(t *testing.T, dir string) {
				os.MkdirAll(PRReviewDir, 0755)
				configPath := filepath.Join(PRReviewDir, "config.json")
				os.WriteFile(configPath, []byte("{}"), 0644)
			},
			expected: true,
		},
		{
			name: "ディレクトリのみ存在（config.jsonなし）",
			setup: func(t *testing.T, dir string) {
				os.MkdirAll(PRReviewDir, 0755)
			},
			expected: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			tt.setup(t, tempDir)

			result := IsInitialized()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCreateDirectory tests directory creation
func TestCreateDirectory(t *testing.T) {
	tests := []struct {
		name        string
		preSetup    func(t *testing.T)
		expectError bool
	}{
		{
			name:        "新規ディレクトリ作成",
			preSetup:    func(t *testing.T) {},
			expectError: false,
		},
		{
			name: "既存ディレクトリが存在",
			preSetup: func(t *testing.T) {
				os.MkdirAll(PRReviewDir, 0755)
			},
			expectError: false,
		},
		{
			name: "ネストされたディレクトリ作成",
			preSetup: func(t *testing.T) {
				// PRReviewDir を一時的に変更してテスト
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			tt.preSetup(t)

			err := CreateDirectory()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify directory exists
			if !tt.expectError {
				if _, err := os.Stat(PRReviewDir); os.IsNotExist(err) {
					t.Error("Directory was not created")
				}
			}
		})
	}
}

// TestUpdateGitignore tests .gitignore updates
func TestUpdateGitignore(t *testing.T) {
	scenarios := []struct {
		name              string
		existingGitignore string
		expectUpdate      bool
		expectedContent   []string
	}{
		{
			name:              ".gitignoreが存在しない",
			existingGitignore: "",
			expectUpdate:      true,
			expectedContent:   []string{".pr-review/"},
		},
		{
			name:              "空の.gitignore",
			existingGitignore: "",
			expectUpdate:      true,
			expectedContent:   []string{".pr-review/"},
		},
		{
			name:              "既に.pr-review/が含まれている",
			existingGitignore: "node_modules/\n.pr-review/\n*.log",
			expectUpdate:      false,
			expectedContent:   []string{"node_modules/", ".pr-review/", "*.log"},
		},
		{
			name:              "他のエントリがある.gitignore",
			existingGitignore: "node_modules/\n*.log\n.env",
			expectUpdate:      true,
			expectedContent:   []string{"node_modules/", "*.log", ".env", ".pr-review/"},
		},
		{
			name:              "コメント付き.gitignore",
			existingGitignore: "# Dependencies\nnode_modules/\n\n# Logs\n*.log",
			expectUpdate:      true,
			expectedContent:   []string{"# Dependencies", "node_modules/", "# Logs", "*.log", ".pr-review/"},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			// Setup existing .gitignore if needed
			if tt.existingGitignore != "" {
				err := os.WriteFile(".gitignore", []byte(tt.existingGitignore), 0644)
				if err != nil {
					t.Fatalf("Failed to create test .gitignore: %v", err)
				}
			}

			err := UpdateGitignore()
			if err != nil {
				t.Errorf("UpdateGitignore failed: %v", err)
			}

			// Check .gitignore content
			content, err := os.ReadFile(".gitignore")
			if err != nil {
				t.Fatalf("Failed to read .gitignore: %v", err)
			}

			contentStr := string(content)
			for _, expected := range tt.expectedContent {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected .gitignore to contain %q, but it doesn't", expected)
				}
			}

			// Verify .pr-review/ is in the file
			if !strings.Contains(contentStr, ".pr-review/") {
				t.Error(".pr-review/ was not added to .gitignore")
			}
		})
	}
}

// TestInitializationWorkflow tests the complete initialization workflow
func TestInitializationWorkflow(t *testing.T) {
	scenarios := []struct {
		name  string
		steps []func() error
	}{
		{
			name: "新規プロジェクトの初期化",
			steps: []func() error{
				CreateDirectory,
				UpdateGitignore,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			// Verify not initialized
			if IsInitialized() {
				t.Error("Should not be initialized at start")
			}

			// Run initialization steps
			for _, step := range scenario.steps {
				if err := step(); err != nil {
					t.Errorf("Step failed: %v", err)
				}
			}

			// Create config.json to complete initialization
			configPath := filepath.Join(PRReviewDir, "config.json")
			os.WriteFile(configPath, []byte("{}"), 0644)

			// Verify initialized
			if !IsInitialized() {
				t.Error("Should be initialized after setup")
			}

			// Verify .gitignore contains .pr-review/
			gitignoreContent, _ := os.ReadFile(".gitignore")
			if !strings.Contains(string(gitignoreContent), ".pr-review/") {
				t.Error(".gitignore should contain .pr-review/")
			}
		})
	}
}

// TestErrorHandling tests error conditions
func TestErrorHandling(t *testing.T) {
	t.Run("読み取り専用ディレクトリ", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()

		// Create read-only parent directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		os.MkdirAll(readOnlyDir, 0555)

		os.Chdir(readOnlyDir)
		defer os.Chdir(originalDir)

		err := CreateDirectory()
		// This might fail on some systems
		if err == nil {
			// If it succeeds, verify the directory exists
			if _, statErr := os.Stat(PRReviewDir); os.IsNotExist(statErr) {
				t.Error("Directory creation reported success but directory doesn't exist")
			}
		}
	})

	t.Run("破損した.gitignore", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Create a .gitignore as a directory (invalid)
		os.MkdirAll(".gitignore", 0755)

		err := UpdateGitignore()
		if err == nil {
			t.Error("Expected error when .gitignore is a directory")
		}
	})
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	done := make(chan bool, 3)

	// Run operations concurrently
	go func() {
		CreateDirectory()
		done <- true
	}()

	go func() {
		UpdateGitignore()
		done <- true
	}()

	go func() {
		IsInitialized()
		done <- true
	}()

	// Wait for all operations
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify final state is consistent
	if _, err := os.Stat(PRReviewDir); os.IsNotExist(err) {
		t.Error("Directory should exist after concurrent operations")
	}
}

// TestIdempotency tests that operations are idempotent
func TestIdempotency(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Run CreateDirectory multiple times
	for i := 0; i < 3; i++ {
		if err := CreateDirectory(); err != nil {
			t.Errorf("CreateDirectory failed on iteration %d: %v", i, err)
		}
	}

	// Run UpdateGitignore multiple times
	for i := 0; i < 3; i++ {
		if err := UpdateGitignore(); err != nil {
			t.Errorf("UpdateGitignore failed on iteration %d: %v", i, err)
		}
	}

	// Verify .pr-review/ appears only once in .gitignore
	content, _ := os.ReadFile(".gitignore")
	count := strings.Count(string(content), ".pr-review/")
	if count != 1 {
		t.Errorf("Expected .pr-review/ to appear once in .gitignore, found %d times", count)
	}
}
