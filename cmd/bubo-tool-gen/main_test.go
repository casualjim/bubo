package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures both zerolog and slog output during test execution
func captureOutput(fn func()) string {
	var buf bytes.Buffer

	// Save old loggers
	oldZeroLogger := log
	oldSlogLogger := slog.Default()
	defer func() {
		log = oldZeroLogger
		slog.SetDefault(oldSlogLogger)
	}()

	// Configure zerolog
	output := zerolog.ConsoleWriter{
		Out:        &buf,
		NoColor:    true,
		TimeFormat: time.Stamp,
	}
	log = zerolog.New(output).With().Timestamp().Logger()

	// Configure slog to use the same zerolog instance
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelDebug}),
	))

	fn()
	return buf.String()
}

func TestCollectTools(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		exportTools bool
		want        []toolFuncInfo
	}{
		{
			name: "single tool function",
			fileContent: `package test
// bubo:agentTool
// This is a test tool
func testTool(param1 string) {}`,
			exportTools: false,
			want: []toolFuncInfo{
				{
					name: "testTool",
					comments: []*ast.Comment{
						{Text: "// This is a test tool"},
					},
					params: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "param1"}},
							Type:  &ast.Ident{Name: "string"},
						},
					},
					exportTools: false,
				},
			},
		},
		{
			name: "multiple tool functions",
			fileContent: `package test
// bubo:agentTool
// Tool 1
func tool1(param1 string) {}

// Not a tool
func notATool() {}

// bubo:agentTool
// Tool 2
func tool2(param1, param2 int) {}`,
			exportTools: true,
			want: []toolFuncInfo{
				{
					name: "tool1",
					comments: []*ast.Comment{
						{Text: "// Tool 1"},
					},
					params: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "param1"}},
							Type:  &ast.Ident{Name: "string"},
						},
					},
					exportTools: true,
				},
				{
					name: "tool2",
					comments: []*ast.Comment{
						{Text: "// Tool 2"},
					},
					params: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "param1"}, {Name: "param2"}},
							Type:  &ast.Ident{Name: "int"},
						},
					},
					exportTools: true,
				},
			},
		},
		{
			name: "no tool functions",
			fileContent: `package test
func regular() {}`,
			exportTools: false,
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "", tt.fileContent, parser.ParseComments)
			require.NoError(t, err)

			got := collectTools(fileAST, tt.exportTools)
			assert.Equal(t, len(tt.want), len(got))

			for i, want := range tt.want {
				assert.Equal(t, want.name, got[i].name)
				assert.Equal(t, len(want.comments), len(got[i].comments))
				for j, comment := range want.comments {
					assert.Equal(t, comment.Text, got[i].comments[j].Text)
				}
				assert.Equal(t, want.exportTools, got[i].exportTools)
			}
		})
	}
}

func TestCreateToolsFile(t *testing.T) {
	tests := []struct {
		name      string
		pkgName   string
		toolFuncs []toolFuncInfo
		wantDecls int
	}{
		{
			name:      "empty tools",
			pkgName:   "test",
			toolFuncs: []toolFuncInfo{},
			wantDecls: 1, // just import declaration
		},
		{
			name:    "single tool",
			pkgName: "test",
			toolFuncs: []toolFuncInfo{
				{
					name: "testTool",
					comments: []*ast.Comment{
						{Text: "// Test tool description"},
					},
					params: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "param1"}},
							Type:  &ast.Ident{Name: "string"},
						},
					},
				},
			},
			wantDecls: 2, // import + 1 tool
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createToolsFile(tt.pkgName, tt.toolFuncs)
			assert.Equal(t, tt.pkgName, got.Name.Name)
			assert.Equal(t, tt.wantDecls, len(got.Decls))
		})
	}
}

func TestProcessGoFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		exportTools bool
		wantErr     bool
		checkFile   bool
	}{
		{
			name: "valid file with tool",
			content: `package test
// bubo:agentTool
// Test tool
func testTool(param string) {}`,
			exportTools: false,
			wantErr:     false,
			checkFile:   true,
		},
		{
			name: "invalid go file",
			content: `package test
invalid go code`,
			exportTools: false,
			wantErr:     true,
			checkFile:   false,
		},
		{
			name: "file without tools",
			content: `package test
func regular() {}`,
			exportTools: false,
			wantErr:     false,
			checkFile:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, tt.name+".go")
			err := os.WriteFile(testFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Capture output and process the file
			output := captureOutput(func() {
				err = processGoFile(testFile, tt.exportTools)
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, output, "Error parsing file")
				return
			}
			assert.NoError(t, err)
			if tt.checkFile {
				assert.Contains(t, output, "Generated file")
			}

			// Check if .bubo.go file was created when expected
			buboFile := filepath.Join(tmpDir, tt.name+".bubo.go")
			if tt.checkFile {
				assert.FileExists(t, buboFile)
				content, err := os.ReadFile(buboFile)
				require.NoError(t, err)
				assert.Contains(t, string(content), "DO NOT EDIT")
			} else {
				_, err := os.Stat(buboFile)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestCreateToolVariableAST(t *testing.T) {
	tests := []struct {
		name     string
		tool     toolFuncInfo
		wantName string
	}{
		{
			name: "basic tool",
			tool: toolFuncInfo{
				name: "testTool",
				comments: []*ast.Comment{
					{Text: "// Test description"},
				},
				params: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: "param1"}},
						Type:  &ast.Ident{Name: "string"},
					},
				},
				exportTools: false,
			},
			wantName: "testToolTool",
		},
		{
			name: "exported tool",
			tool: toolFuncInfo{
				name: "testTool",
				comments: []*ast.Comment{
					{Text: "// Test description"},
				},
				exportTools: true,
			},
			wantName: "TestToolTool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decl := createToolVariableAST(tt.tool)
			genDecl, ok := decl.(*ast.GenDecl)
			require.True(t, ok)
			assert.Equal(t, token.VAR, genDecl.Tok)

			spec, ok := genDecl.Specs[0].(*ast.ValueSpec)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, spec.Names[0].Name)

			// Verify comments are preserved
			if len(tt.tool.comments) > 0 {
				assert.Equal(t, tt.tool.comments[0].Text, genDecl.Doc.List[0].Text)
			}
		})
	}
}

func TestMainFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files in separate directories to avoid cross-contamination
	validDir := filepath.Join(tmpDir, "valid")
	require.NoError(t, os.MkdirAll(validDir, 0o755))

	validFile := filepath.Join(validDir, "valid.go")
	err := os.WriteFile(validFile, []byte(`package test
// bubo:agentTool
// Test tool
func testTool(param string) {}`), 0o644)
	require.NoError(t, err)

	invalidDir := filepath.Join(tmpDir, "invalid")
	require.NoError(t, os.MkdirAll(invalidDir, 0o755))

	invalidFile := filepath.Join(invalidDir, "invalid.go")
	err = os.WriteFile(invalidFile, []byte("invalid go code"), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "process directory",
			args:    []string{"-path", validDir},
			wantErr: false,
		},
		{
			name:    "process single valid file",
			args:    []string{"-path", validFile},
			wantErr: false,
		},
		{
			name:    "process single invalid file",
			args:    []string{"-path", invalidFile},
			wantErr: true,
		},
		{
			name:    "invalid path",
			args:    []string{"-path", "/nonexistent/path"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test environment
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			os.Args = append([]string{"cmd"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			t.Logf("Running test case: %s with args: %v", tt.name, tt.args)

			// Mock os.Exit
			var exitCode int
			oldOsExit := osExit
			defer func() { osExit = oldOsExit }()
			osExit = func(code int) {
				exitCode = code
				panic(fmt.Sprintf("os.Exit(%d)", code))
			}

			output := captureOutput(func() {
				defer func() {
					if r := recover(); r != nil {
						// Expected panic from os.Exit
						t.Logf("Recovered from panic: %v", r)
					}
				}()
				main()
			})

			t.Logf("Captured output: %s", output)
			t.Logf("Exit code: %d, Want error: %v", exitCode, tt.wantErr)

			if tt.wantErr {
				assert.Equal(t, 1, exitCode, "Expected exit code 1 for error case")
			} else {
				assert.Equal(t, 0, exitCode, "Expected exit code 0 for success case")
			}

			// Verify output based on test case
			switch tt.name {
			case "process directory", "process single valid file":
				if !tt.wantErr {
					assert.Contains(t, output, "Generated file",
						"Expected 'Generated file' in output for success case")
				}
			case "process single invalid file":
				if tt.wantErr {
					assert.Contains(t, output, "Error parsing file",
						"Expected 'Error parsing file' in output for invalid file")
				}
			case "invalid path":
				if tt.wantErr {
					assert.Contains(t, output, "Error accessing path",
						"Expected 'Error accessing path' in output for invalid path")
				}
			}
		})
	}
}
