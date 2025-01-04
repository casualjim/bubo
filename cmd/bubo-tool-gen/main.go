package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/casualjim/bubo/pkg/slogx"
	"github.com/go-openapi/swag"
	"github.com/phsym/zeroslog"
	"github.com/rs/zerolog"
)

var (
	log    zerolog.Logger
	osExit = os.Exit
)

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log = zerolog.New(output).With().Timestamp().Logger()
}

func init() {
	slog.SetDefault(slog.New(
		zeroslog.NewHandler(log, &zeroslog.HandlerOptions{Level: slog.LevelDebug}),
	))
}

func processGoFile(path string, exportTools bool) error {
	fset := token.NewFileSet()

	// Parse the Go file
	fileAST, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		slog.Error("Error parsing file", slog.String("path", path), slogx.Error(err))
		return err
	}

	// Get package name and collect agent tools
	pkgName := fileAST.Name.Name
	toolFuncs := collectTools(fileAST, exportTools)

	if len(toolFuncs) == 0 {
		return nil
	}

	// Create new file AST for tool declarations
	outputPath := strings.TrimSuffix(path, ".go") + ".bubo.go"
	newFileAST := createToolsFile(pkgName, toolFuncs)

	// Write the new file
	var buf bytes.Buffer
	cfg := printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}
	err = cfg.Fprint(&buf, fset, newFileAST)
	if err != nil {
		slog.Error("Error writing file", slog.String("path", outputPath), slogx.Error(err))
		return err
	}

	// Add extra newlines between declarations and before comment blocks
	output := buf.String()
	lines := strings.Split(output, "\n")
	formattedLines := []string{
		"// Code generated by bubo-tool-gen. DO NOT EDIT.",
		"// This file was generated for bubo:agentTool marker comments in the source code.",
		"",
	}

	for i, line := range slices.All(lines) {
		// Add newline before the first line of a comment block
		if strings.HasPrefix(line, "//") && i > 0 {
			// Check if this is the first line of a comment block
			prevLine := strings.TrimSpace(lines[i-1])
			if !strings.HasPrefix(prevLine, "//") && !strings.Contains(prevLine, "package") {
				formattedLines = append(formattedLines, "")
			}
		}
		formattedLines = append(formattedLines, line)
		// Add newline after var declaration
		if strings.HasPrefix(line, "var") {
			formattedLines = append(formattedLines, "")
		}
	}

	err = os.WriteFile(outputPath, []byte(strings.Join(formattedLines, "\n")), 0o600)
	if err != nil {
		slog.Error("Error writing file", slog.String("path", outputPath), slogx.Error(err))
		return err
	}

	slog.Info("Generated file", slog.String("path", outputPath))
	return nil
}

func main() {
	targetPath := flag.String("path", "./", "Path to file or directory to process")
	exportTools := flag.Bool("export", false, "Ensure the tool definitions are exported")
	flag.Parse()

	// Get file info for the target path
	fileInfo, err := os.Stat(*targetPath)
	if err != nil {
		slog.Error("Error accessing path", slog.String("path", *targetPath), slogx.Error(err))
		osExit(1)
	}

	if fileInfo.IsDir() {
		// Walk through the directory and process Go files
		err = filepath.Walk(*targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories, non-Go files, and .bubo.go files
			if info.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".bubo.go") {
				return nil
			}

			return processGoFile(path, *exportTools)
		})
		if err != nil {
			slog.Error("Error walking the path", slog.String("dir", *targetPath), slogx.Error(err))
			osExit(1)
		}
	} else {
		// Process single file
		if !strings.HasSuffix(*targetPath, ".go") || strings.HasSuffix(*targetPath, "_test.go") {
			slog.Error("Not a Go source file", slog.String("path", *targetPath))
			osExit(1)
		}

		if err := processGoFile(*targetPath, *exportTools); err != nil {
			osExit(1)
		}
	}
}

type toolFuncInfo struct {
	name        string
	comments    []*ast.Comment
	params      []*ast.Field
	exportTools bool
}

func collectTools(fileAST *ast.File, exportTools bool) []toolFuncInfo {
	var toolFuncs []toolFuncInfo

	for decl := range slices.Values(fileAST.Decls) {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Doc != nil {
			var comments []*ast.Comment
			isToolFunc := false
			for comment := range slices.Values(funcDecl.Doc.List) {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
				if text == "bubo:agentTool" {
					isToolFunc = true
				} else if text != "" { // Only include non-empty comments
					comments = append(comments, comment)
				}
			}
			if isToolFunc {
				toolFuncs = append(toolFuncs, toolFuncInfo{
					name:        funcDecl.Name.Name,
					comments:    comments,
					params:      funcDecl.Type.Params.List,
					exportTools: exportTools,
				})
			}
		}
	}

	return toolFuncs
}

func createToolsFile(pkgName string, toolFuncs []toolFuncInfo) *ast.File {
	// Create new file AST
	newFile := &ast.File{
		Name: ast.NewIdent(pkgName),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/casualjim/bubo/tool"`,
						},
					},
				},
			},
		},
	}

	// Add tool variable declarations
	for tool := range slices.Values(toolFuncs) {
		toolDecl := createToolVariableAST(tool)
		newFile.Decls = append(newFile.Decls, toolDecl)
	}

	return newFile
}

func createToolVariableAST(tool toolFuncInfo) ast.Decl {
	// Get parameter names
	var paramExprs []ast.Expr
	for field := range slices.Values(tool.params) {
		for name := range slices.Values(field.Names) {
			paramExprs = append(paramExprs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", name.Name),
			})
		}
	}

	// Get tool description from comments
	description := ""
	for comment := range slices.Values(tool.comments) {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if description == "" {
			description = text
		} else {
			description += " " + text
		}
	}

	// Create the arguments for MustAgentTool
	args := []ast.Expr{
		ast.NewIdent(tool.name),
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("tool"),
				Sel: ast.NewIdent("Name"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf("%q", strings.TrimSpace(tool.name)),
				},
			},
		},
	}

	fnDesc := description
	if strings.TrimSpace(fnDesc) == "" {
		fnDesc = swag.ToHumanNameLower(tool.name)
	}
	args = append(args, &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("tool"),
			Sel: ast.NewIdent("Description"),
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", strings.TrimSpace(fnDesc)),
			},
		},
	})

	// Only add WithToolParameters if there are parameters
	if len(paramExprs) > 0 {
		args = append(args, &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("tool"),
				Sel: ast.NewIdent("Parameters"),
			},
			Args: paramExprs,
		})
	}

	// Create the MustAgentTool call expression
	callExpr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("tool"),
			Sel: ast.NewIdent("Must"),
		},
		Args: args,
	}

	// Create the variable declaration
	toolName := tool.name
	if tool.exportTools {
		toolName = swag.ToGoName(toolName)
	}

	// Create comment group from original comments
	var commentList []*ast.Comment //nolint:prealloc
	for comment := range slices.Values(tool.comments) {
		commentList = append(commentList, &ast.Comment{Text: comment.Text})
	}

	genDecl := &ast.GenDecl{
		Doc: &ast.CommentGroup{List: commentList},
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					ast.NewIdent(toolName + "Tool"),
				},
				Values: []ast.Expr{
					callExpr,
				},
			},
		},
	}

	return genDecl
}
