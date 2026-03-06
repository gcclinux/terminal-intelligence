package docgen

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ProjectAnalyzer scans and analyzes project files
type ProjectAnalyzer struct {
	workspaceRoot string
	scopeFilters  []string
	gitignore     *GitignoreParser
}

// NewProjectAnalyzer creates a new ProjectAnalyzer
func NewProjectAnalyzer(workspaceRoot string, scopeFilters []string) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		workspaceRoot: workspaceRoot,
		scopeFilters:  scopeFilters,
		gitignore:     NewGitignoreParser(workspaceRoot),
	}
}

// DiscoveredFiles represents files discovered during scanning
type DiscoveredFiles struct {
	CodeFiles   []string
	ConfigFiles []string
	DocFiles    []string
	AllFiles    []string
}

// DiscoverFiles scans the workspace and returns discovered files by type
func (a *ProjectAnalyzer) DiscoverFiles() (*DiscoveredFiles, error) {
	result := &DiscoveredFiles{
		CodeFiles:   make([]string, 0),
		ConfigFiles: make([]string, 0),
		DocFiles:    make([]string, 0),
		AllFiles:    make([]string, 0),
	}

	// Skip directories
	skipDirs := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".git":         true,
		"build":        true,
		"dist":         true,
	}

	err := filepath.WalkDir(a.workspaceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log warning but continue
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(a.workspaceRoot, path)
		if err != nil {
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check if directory should be skipped
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check gitignore
		if a.gitignore.IsIgnored(relPath) {
			return nil
		}

		// Apply scope filters if present
		if len(a.scopeFilters) > 0 && !a.matchesScopeFilter(relPath) {
			return nil
		}

		// Categorize file by type
		result.AllFiles = append(result.AllFiles, relPath)

		if a.isCodeFile(relPath) {
			result.CodeFiles = append(result.CodeFiles, relPath)
		} else if a.isConfigFile(relPath) {
			result.ConfigFiles = append(result.ConfigFiles, relPath)
		} else if a.isDocFile(relPath) {
			result.DocFiles = append(result.DocFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return result, nil
}

// isCodeFile checks if a file is a code file
func (a *ProjectAnalyzer) isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go":    true,
		".py":    true,
		".js":    true,
		".ts":    true,
		".jsx":   true,
		".tsx":   true,
		".java":  true,
		".c":     true,
		".cpp":   true,
		".h":     true,
		".hpp":   true,
		".rs":    true,
		".rb":    true,
		".php":   true,
		".cs":    true,
		".swift": true,
		".kt":    true,
	}
	return codeExts[ext]
}

// isConfigFile checks if a file is a configuration file
func (a *ProjectAnalyzer) isConfigFile(path string) bool {
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	// Check by filename
	configFiles := map[string]bool{
		"go.mod":            true,
		"go.sum":            true,
		"package.json":      true,
		"package-lock.json": true,
		"requirements.txt":  true,
		"Pipfile":           true,
		"Cargo.toml":        true,
		"Cargo.lock":        true,
		"Makefile":          true,
		"Dockerfile":        true,
		".env":              true,
		".gitignore":        true,
	}

	if configFiles[base] {
		return true
	}

	// Check by extension
	configExts := map[string]bool{
		".yaml": true,
		".yml":  true,
		".json": true,
		".toml": true,
		".ini":  true,
		".conf": true,
	}

	return configExts[ext]
}

// isDocFile checks if a file is a documentation file
func (a *ProjectAnalyzer) isDocFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	// Check by filename
	if strings.HasPrefix(base, "readme") {
		return true
	}

	docFiles := map[string]bool{
		"changelog.md":    true,
		"changelog":       true,
		"contributing.md": true,
		"license":         true,
		"license.md":      true,
		"authors":         true,
		"authors.md":      true,
	}

	if docFiles[base] {
		return true
	}

	// Check by extension
	docExts := map[string]bool{
		".md":  true,
		".txt": true,
		".rst": true,
	}

	return docExts[ext]
}

// matchesScopeFilter checks if a path matches any scope filter
func (a *ProjectAnalyzer) matchesScopeFilter(path string) bool {
	for _, filter := range a.scopeFilters {
		// Simple pattern matching - check if path contains filter
		if strings.Contains(path, filter) {
			return true
		}

		// Check if filter is a directory prefix
		if strings.HasPrefix(path, filter) {
			return true
		}

		// Check if filter matches filename
		if filepath.Base(path) == filter {
			return true
		}
	}
	return false
}

// GitignoreParser handles .gitignore pattern matching
type GitignoreParser struct {
	patterns []string
}

// NewGitignoreParser creates a new GitignoreParser
func NewGitignoreParser(workspaceRoot string) *GitignoreParser {
	parser := &GitignoreParser{
		patterns: make([]string, 0),
	}

	gitignorePath := filepath.Join(workspaceRoot, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		// No .gitignore file, return empty parser
		return parser
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parser.patterns = append(parser.patterns, line)
	}

	return parser
}

// IsIgnored checks if a path should be ignored based on .gitignore patterns
func (g *GitignoreParser) IsIgnored(path string) bool {
	for _, pattern := range g.patterns {
		if g.matchPattern(pattern, path) {
			return true
		}
	}
	return false
}

// matchPattern performs simple pattern matching for .gitignore patterns
func (g *GitignoreParser) matchPattern(pattern, path string) bool {
	// Remove leading slash
	pattern = strings.TrimPrefix(pattern, "/")

	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		// Check if path starts with this directory
		return strings.HasPrefix(path, pattern+string(filepath.Separator)) || path == pattern
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		if pattern == "*" {
			return true
		}

		// Handle *.ext patterns
		if strings.HasPrefix(pattern, "*.") {
			ext := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(path, ext)
		}

		// Handle prefix* patterns
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(filepath.Base(path), prefix)
		}
	}

	// Exact match or basename match
	return path == pattern || filepath.Base(path) == pattern
}

// AnalyzeGoFile analyzes a Go source file and extracts code structure
func (a *ProjectAnalyzer) AnalyzeGoFile(filePath string) (*CodeStructure, error) {
	fullPath := filepath.Join(a.workspaceRoot, filePath)

	// Create a new token file set
	fset := token.NewFileSet()

	// Parse the file
	file, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filePath, err)
	}

	structure := &CodeStructure{
		Packages:   make([]PackageInfo, 0),
		Functions:  make([]FunctionInfo, 0),
		Classes:    make([]ClassInfo, 0),
		Structs:    make([]StructInfo, 0),
		Interfaces: make([]InterfaceInfo, 0),
		Exports:    make([]ExportInfo, 0),
	}

	// Extract package information
	pkgInfo := PackageInfo{
		Name:        file.Name.Name,
		Path:        filePath,
		Description: extractPackageComment(file),
	}
	structure.Packages = append(structure.Packages, pkgInfo)

	// Walk through declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			funcInfo := a.extractFunctionInfo(d, pkgInfo.Name)
			structure.Functions = append(structure.Functions, funcInfo)

			// Add to exports if exported
			if funcInfo.IsExported {
				structure.Exports = append(structure.Exports, ExportInfo{
					Name:    funcInfo.Name,
					Type:    "function",
					Package: pkgInfo.Name,
				})
			}

		case *ast.GenDecl:
			a.extractGenDeclInfo(d, pkgInfo.Name, structure)
		}
	}

	return structure, nil
}

// extractPackageComment extracts the package-level comment
func extractPackageComment(file *ast.File) string {
	if file.Doc != nil {
		return strings.TrimSpace(file.Doc.Text())
	}
	// If no package doc, check for comments before package declaration
	if len(file.Comments) > 0 {
		// The first comment group might be the package comment
		firstComment := file.Comments[0]
		if firstComment.Pos() < file.Package {
			return strings.TrimSpace(firstComment.Text())
		}
	}
	return ""
}

// extractFunctionInfo extracts information from a function declaration
func (a *ProjectAnalyzer) extractFunctionInfo(funcDecl *ast.FuncDecl, pkgName string) FunctionInfo {
	funcInfo := FunctionInfo{
		Name:       funcDecl.Name.Name,
		Package:    pkgName,
		Parameters: make([]Parameter, 0),
		Returns:    make([]ReturnValue, 0),
		IsExported: ast.IsExported(funcDecl.Name.Name),
	}

	// Extract comment
	if funcDecl.Doc != nil {
		funcInfo.Comment = funcDecl.Doc.Text()
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			typeStr := a.exprToString(field.Type)
			if len(field.Names) == 0 {
				// Unnamed parameter
				funcInfo.Parameters = append(funcInfo.Parameters, Parameter{
					Name: "",
					Type: typeStr,
				})
			} else {
				// Named parameters
				for _, name := range field.Names {
					funcInfo.Parameters = append(funcInfo.Parameters, Parameter{
						Name: name.Name,
						Type: typeStr,
					})
				}
			}
		}
	}

	// Extract return values
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			typeStr := a.exprToString(field.Type)
			if len(field.Names) == 0 {
				// Unnamed return value
				funcInfo.Returns = append(funcInfo.Returns, ReturnValue{
					Name: "",
					Type: typeStr,
				})
			} else {
				// Named return values
				for _, name := range field.Names {
					funcInfo.Returns = append(funcInfo.Returns, ReturnValue{
						Name: name.Name,
						Type: typeStr,
					})
				}
			}
		}
	}

	// Build signature
	funcInfo.Signature = a.buildFunctionSignature(funcInfo)

	return funcInfo
}

// extractGenDeclInfo extracts information from a general declaration (type, const, var)
func (a *ProjectAnalyzer) extractGenDeclInfo(genDecl *ast.GenDecl, pkgName string, structure *CodeStructure) {
	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			switch t := s.Type.(type) {
			case *ast.StructType:
				structInfo := a.extractStructInfo(s, t, genDecl.Doc, pkgName)
				structure.Structs = append(structure.Structs, structInfo)

				// Add to exports if exported
				if structInfo.IsExported {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    structInfo.Name,
						Type:    "struct",
						Package: pkgName,
					})
				}

			case *ast.InterfaceType:
				interfaceInfo := a.extractInterfaceInfo(s, t, genDecl.Doc, pkgName)
				structure.Interfaces = append(structure.Interfaces, interfaceInfo)

				// Add to exports if exported
				if interfaceInfo.IsExported {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    interfaceInfo.Name,
						Type:    "interface",
						Package: pkgName,
					})
				}

			default:
				// Other type declarations (type aliases, etc.)
				if ast.IsExported(s.Name.Name) {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    s.Name.Name,
						Type:    "type",
						Package: pkgName,
					})
				}
			}

		case *ast.ValueSpec:
			// Constants and variables
			for _, name := range s.Names {
				if ast.IsExported(name.Name) {
					exportType := "variable"
					if genDecl.Tok == token.CONST {
						exportType = "constant"
					}
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    name.Name,
						Type:    exportType,
						Package: pkgName,
					})
				}
			}
		}
	}
}

// extractStructInfo extracts information from a struct type
func (a *ProjectAnalyzer) extractStructInfo(typeSpec *ast.TypeSpec, structType *ast.StructType, doc *ast.CommentGroup, pkgName string) StructInfo {
	structInfo := StructInfo{
		Name:       typeSpec.Name.Name,
		Package:    pkgName,
		Fields:     make([]FieldInfo, 0),
		Methods:    make([]FunctionInfo, 0),
		IsExported: ast.IsExported(typeSpec.Name.Name),
	}

	// Extract comment
	if typeSpec.Doc != nil {
		structInfo.Comment = typeSpec.Doc.Text()
	} else if doc != nil {
		structInfo.Comment = doc.Text()
	}

	// Extract fields
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			typeStr := a.exprToString(field.Type)
			tagStr := ""
			if field.Tag != nil {
				tagStr = field.Tag.Value
			}

			fieldComment := ""
			if field.Doc != nil {
				fieldComment = field.Doc.Text()
			} else if field.Comment != nil {
				fieldComment = field.Comment.Text()
			}

			if len(field.Names) == 0 {
				// Embedded field
				structInfo.Fields = append(structInfo.Fields, FieldInfo{
					Name:    typeStr,
					Type:    typeStr,
					Tag:     tagStr,
					Comment: fieldComment,
				})
			} else {
				// Named fields
				for _, name := range field.Names {
					structInfo.Fields = append(structInfo.Fields, FieldInfo{
						Name:    name.Name,
						Type:    typeStr,
						Tag:     tagStr,
						Comment: fieldComment,
					})
				}
			}
		}
	}

	return structInfo
}

// extractInterfaceInfo extracts information from an interface type
func (a *ProjectAnalyzer) extractInterfaceInfo(typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType, doc *ast.CommentGroup, pkgName string) InterfaceInfo {
	interfaceInfo := InterfaceInfo{
		Name:       typeSpec.Name.Name,
		Package:    pkgName,
		Methods:    make([]MethodSignature, 0),
		IsExported: ast.IsExported(typeSpec.Name.Name),
	}

	// Extract comment
	if typeSpec.Doc != nil {
		interfaceInfo.Comment = typeSpec.Doc.Text()
	} else if doc != nil {
		interfaceInfo.Comment = doc.Text()
	}

	// Extract methods
	if interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if len(method.Names) > 0 {
				// Method signature
				methodSig := MethodSignature{
					Name:       method.Names[0].Name,
					Parameters: make([]Parameter, 0),
					Returns:    make([]ReturnValue, 0),
				}

				// Extract method type
				if funcType, ok := method.Type.(*ast.FuncType); ok {
					// Extract parameters
					if funcType.Params != nil {
						for _, field := range funcType.Params.List {
							typeStr := a.exprToString(field.Type)
							if len(field.Names) == 0 {
								methodSig.Parameters = append(methodSig.Parameters, Parameter{
									Name: "",
									Type: typeStr,
								})
							} else {
								for _, name := range field.Names {
									methodSig.Parameters = append(methodSig.Parameters, Parameter{
										Name: name.Name,
										Type: typeStr,
									})
								}
							}
						}
					}

					// Extract return values
					if funcType.Results != nil {
						for _, field := range funcType.Results.List {
							typeStr := a.exprToString(field.Type)
							if len(field.Names) == 0 {
								methodSig.Returns = append(methodSig.Returns, ReturnValue{
									Name: "",
									Type: typeStr,
								})
							} else {
								for _, name := range field.Names {
									methodSig.Returns = append(methodSig.Returns, ReturnValue{
										Name: name.Name,
										Type: typeStr,
									})
								}
							}
						}
					}
				}

				interfaceInfo.Methods = append(interfaceInfo.Methods, methodSig)
			}
		}
	}

	return interfaceInfo
}

// exprToString converts an AST expression to a string representation
func (a *ProjectAnalyzer) exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name

	case *ast.SelectorExpr:
		return a.exprToString(e.X) + "." + e.Sel.Name

	case *ast.StarExpr:
		return "*" + a.exprToString(e.X)

	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + a.exprToString(e.Elt)
		}
		return "[" + a.exprToString(e.Len) + "]" + a.exprToString(e.Elt)

	case *ast.MapType:
		return "map[" + a.exprToString(e.Key) + "]" + a.exprToString(e.Value)

	case *ast.ChanType:
		switch e.Dir {
		case ast.SEND:
			return "chan<- " + a.exprToString(e.Value)
		case ast.RECV:
			return "<-chan " + a.exprToString(e.Value)
		default:
			return "chan " + a.exprToString(e.Value)
		}

	case *ast.FuncType:
		return a.funcTypeToString(e)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.StructType:
		return "struct{}"

	case *ast.Ellipsis:
		return "..." + a.exprToString(e.Elt)

	case *ast.BasicLit:
		return e.Value

	default:
		return fmt.Sprintf("%T", expr)
	}
}

// funcTypeToString converts a function type to a string
func (a *ProjectAnalyzer) funcTypeToString(funcType *ast.FuncType) string {
	var sb strings.Builder
	sb.WriteString("func(")

	// Parameters
	if funcType.Params != nil {
		for i, field := range funcType.Params.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			typeStr := a.exprToString(field.Type)
			if len(field.Names) > 0 {
				for j, name := range field.Names {
					if j > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(name.Name)
					sb.WriteString(" ")
				}
			}
			sb.WriteString(typeStr)
		}
	}

	sb.WriteString(")")

	// Return values
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		sb.WriteString(" ")
		if len(funcType.Results.List) > 1 {
			sb.WriteString("(")
		}
		for i, field := range funcType.Results.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			typeStr := a.exprToString(field.Type)
			if len(field.Names) > 0 {
				for j, name := range field.Names {
					if j > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(name.Name)
					sb.WriteString(" ")
				}
			}
			sb.WriteString(typeStr)
		}
		if len(funcType.Results.List) > 1 {
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// buildFunctionSignature builds a complete function signature string
func (a *ProjectAnalyzer) buildFunctionSignature(funcInfo FunctionInfo) string {
	var sb strings.Builder
	sb.WriteString("func ")
	sb.WriteString(funcInfo.Name)
	sb.WriteString("(")

	// Parameters
	for i, param := range funcInfo.Parameters {
		if i > 0 {
			sb.WriteString(", ")
		}
		if param.Name != "" {
			sb.WriteString(param.Name)
			sb.WriteString(" ")
		}
		sb.WriteString(param.Type)
	}

	sb.WriteString(")")

	// Return values
	if len(funcInfo.Returns) > 0 {
		sb.WriteString(" ")
		if len(funcInfo.Returns) > 1 {
			sb.WriteString("(")
		}
		for i, ret := range funcInfo.Returns {
			if i > 0 {
				sb.WriteString(", ")
			}
			if ret.Name != "" {
				sb.WriteString(ret.Name)
				sb.WriteString(" ")
			}
			sb.WriteString(ret.Type)
		}
		if len(funcInfo.Returns) > 1 {
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// AnalyzePythonFile analyzes a Python source file and extracts code structure
func (a *ProjectAnalyzer) AnalyzePythonFile(filePath string) (*CodeStructure, error) {
	fullPath := filepath.Join(a.workspaceRoot, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Python file %s: %w", filePath, err)
	}

	structure := &CodeStructure{
		Packages:   make([]PackageInfo, 0),
		Functions:  make([]FunctionInfo, 0),
		Classes:    make([]ClassInfo, 0),
		Structs:    make([]StructInfo, 0),
		Interfaces: make([]InterfaceInfo, 0),
		Exports:    make([]ExportInfo, 0),
	}

	lines := strings.Split(string(content), "\n")

	// Extract module-level docstring as package description
	pkgDesc := extractPythonModuleDocstring(lines)
	pkgInfo := PackageInfo{
		Name:        strings.TrimSuffix(filepath.Base(filePath), ".py"),
		Path:        filePath,
		Description: pkgDesc,
	}
	structure.Packages = append(structure.Packages, pkgInfo)

	// Extract functions and classes
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Extract class definitions
		if strings.HasPrefix(line, "class ") {
			classInfo, nextLine := extractPythonClass(lines, i, pkgInfo.Name)
			if classInfo != nil {
				structure.Classes = append(structure.Classes, *classInfo)

				// Add to exports if not private (doesn't start with _)
				if !strings.HasPrefix(classInfo.Name, "_") {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    classInfo.Name,
						Type:    "class",
						Package: pkgInfo.Name,
					})
				}
			}
			i = nextLine - 1
			continue
		}

		// Extract top-level function definitions (not indented)
		if strings.HasPrefix(line, "def ") {
			funcInfo, nextLine := extractPythonFunction(lines, i, pkgInfo.Name, 0)
			if funcInfo != nil {
				structure.Functions = append(structure.Functions, *funcInfo)

				// Add to exports if not private (doesn't start with _)
				if !strings.HasPrefix(funcInfo.Name, "_") {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    funcInfo.Name,
						Type:    "function",
						Package: pkgInfo.Name,
					})
				}
			}
			i = nextLine - 1
		}
	}

	return structure, nil
}

// Analyze performs a complete analysis of the project and returns AnalysisResult
func (a *ProjectAnalyzer) Analyze() (*AnalysisResult, error) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Packages:   make([]PackageInfo, 0),
			Functions:  make([]FunctionInfo, 0),
			Classes:    make([]ClassInfo, 0),
			Structs:    make([]StructInfo, 0),
			Interfaces: make([]InterfaceInfo, 0),
			Exports:    make([]ExportInfo, 0),
		},
		Configuration: &ConfigInfo{
			PackageManifests: make([]string, 0),
			BuildScripts:     make([]string, 0),
			ConfigFiles:      make([]string, 0),
		},
		Documentation: &ExistingDocs{
			ReadmeContent: "",
			Comments:      make([]CommentBlock, 0),
			Docstrings:    make([]DocstringBlock, 0),
		},
		Dependencies: &DependencyInfo{
			Runtime: make([]Dependency, 0),
			Build:   make([]Dependency, 0),
		},
	}

	// Discover files
	files, err := a.DiscoverFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	// Process code files
	for _, codeFile := range files.CodeFiles {
		ext := strings.ToLower(filepath.Ext(codeFile))
		var structure *CodeStructure
		var analyzeErr error

		switch ext {
		case ".go":
			structure, analyzeErr = a.AnalyzeGoFile(codeFile)
		case ".py":
			structure, analyzeErr = a.AnalyzePythonFile(codeFile)
		case ".js", ".ts", ".jsx", ".tsx":
			structure, analyzeErr = a.AnalyzeJavaScriptFile(codeFile)
		default:
			// Skip unsupported file types
			continue
		}

		if analyzeErr != nil {
			// Log warning but continue with other files
			continue
		}

		// Merge code structure
		if structure != nil {
			result.CodeStructure.Packages = append(result.CodeStructure.Packages, structure.Packages...)
			result.CodeStructure.Functions = append(result.CodeStructure.Functions, structure.Functions...)
			result.CodeStructure.Classes = append(result.CodeStructure.Classes, structure.Classes...)
			result.CodeStructure.Structs = append(result.CodeStructure.Structs, structure.Structs...)
			result.CodeStructure.Interfaces = append(result.CodeStructure.Interfaces, structure.Interfaces...)
			result.CodeStructure.Exports = append(result.CodeStructure.Exports, structure.Exports...)
		}
	}

	// Process configuration files
	for _, configFile := range files.ConfigFiles {
		base := filepath.Base(configFile)

		// Identify package manifests
		if a.isPackageManifest(base) {
			result.Configuration.PackageManifests = append(result.Configuration.PackageManifests, configFile)

			// Extract dependencies
			deps, err := a.extractDependencies(configFile)
			if err == nil && deps != nil {
				result.Dependencies.Runtime = append(result.Dependencies.Runtime, deps.Runtime...)
				result.Dependencies.Build = append(result.Dependencies.Build, deps.Build...)
			}
		}

		// Identify build scripts
		if a.isBuildScript(base) {
			result.Configuration.BuildScripts = append(result.Configuration.BuildScripts, configFile)
		}

		// Add to config files list
		result.Configuration.ConfigFiles = append(result.Configuration.ConfigFiles, configFile)
	}

	// Process documentation files
	for _, docFile := range files.DocFiles {
		base := strings.ToLower(filepath.Base(docFile))

		// Extract README content
		if strings.HasPrefix(base, "readme") {
			content, err := a.readFileContent(docFile)
			if err == nil {
				result.Documentation.ReadmeContent = content
			}
		}
	}

	return result, nil
}

// isPackageManifest checks if a file is a package manifest
func (a *ProjectAnalyzer) isPackageManifest(filename string) bool {
	manifests := map[string]bool{
		"go.mod":           true,
		"package.json":     true,
		"requirements.txt": true,
		"Pipfile":          true,
		"Cargo.toml":       true,
	}
	return manifests[filename]
}

// isBuildScript checks if a file is a build script
func (a *ProjectAnalyzer) isBuildScript(filename string) bool {
	scripts := map[string]bool{
		"Makefile":   true,
		"build.sh":   true,
		"build.bat":  true,
		"Dockerfile": true,
	}
	return scripts[filename]
}

// readFileContent reads the content of a file
func (a *ProjectAnalyzer) readFileContent(filePath string) (string, error) {
	fullPath := filepath.Join(a.workspaceRoot, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(content), nil
}

// extractDependencies extracts dependencies from a package manifest file
func (a *ProjectAnalyzer) extractDependencies(filePath string) (*DependencyInfo, error) {
	base := filepath.Base(filePath)
	content, err := a.readFileContent(filePath)
	if err != nil {
		return nil, err
	}

	deps := &DependencyInfo{
		Runtime: make([]Dependency, 0),
		Build:   make([]Dependency, 0),
	}

	switch base {
	case "go.mod":
		deps.Runtime = a.extractGoModDependencies(content, base)
	case "package.json":
		deps.Runtime = a.extractPackageJsonDependencies(content, base)
	case "requirements.txt":
		deps.Runtime = a.extractRequirementsTxtDependencies(content, base)
	case "Pipfile":
		deps.Runtime = a.extractPipfileDependencies(content, base)
	case "Cargo.toml":
		deps.Runtime = a.extractCargoTomlDependencies(content, base)
	}

	return deps, nil
}

// extractGoModDependencies extracts dependencies from go.mod
func (a *ProjectAnalyzer) extractGoModDependencies(content, source string) []Dependency {
	deps := make([]Dependency, 0)
	lines := strings.Split(content, "\n")

	inRequireBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock || strings.HasPrefix(line, "require ") {
			// Parse dependency line
			line = strings.TrimPrefix(line, "require ")
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				version := parts[1]
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Source:  source,
				})
			}
		}
	}

	return deps
}

// extractPackageJsonDependencies extracts dependencies from package.json
func (a *ProjectAnalyzer) extractPackageJsonDependencies(content, source string) []Dependency {
	deps := make([]Dependency, 0)
	lines := strings.Split(content, "\n")

	inDepsBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, `"dependencies"`) || strings.Contains(trimmed, `"devDependencies"`) {
			inDepsBlock = true
			continue
		}

		if inDepsBlock && (trimmed == "}" || trimmed == "},") {
			inDepsBlock = false
			continue
		}

		if inDepsBlock && strings.Contains(trimmed, ":") {
			// Parse dependency line: "name": "version",
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[0]), `"`)
				version := strings.Trim(strings.TrimSpace(parts[1]), `",`)
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Source:  source,
				})
			}
		}
	}

	return deps
}

// extractRequirementsTxtDependencies extracts dependencies from requirements.txt
func (a *ProjectAnalyzer) extractRequirementsTxtDependencies(content, source string) []Dependency {
	deps := make([]Dependency, 0)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse dependency line: package==version or package>=version
		var name, version string
		if strings.Contains(line, "==") {
			parts := strings.SplitN(line, "==", 2)
			name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, ">=") {
			parts := strings.SplitN(line, ">=", 2)
			name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				version = ">=" + strings.TrimSpace(parts[1])
			}
		} else {
			name = line
			version = ""
		}

		if name != "" {
			deps = append(deps, Dependency{
				Name:    name,
				Version: version,
				Source:  source,
			})
		}
	}

	return deps
}

// extractPipfileDependencies extracts dependencies from Pipfile
func (a *ProjectAnalyzer) extractPipfileDependencies(content, source string) []Dependency {
	deps := make([]Dependency, 0)
	lines := strings.Split(content, "\n")

	inPackagesBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[packages]" {
			inPackagesBlock = true
			continue
		}

		if inPackagesBlock && strings.HasPrefix(trimmed, "[") {
			inPackagesBlock = false
			continue
		}

		if inPackagesBlock && strings.Contains(trimmed, "=") {
			// Parse dependency line: package = "version"
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Source:  source,
				})
			}
		}
	}

	return deps
}

// extractCargoTomlDependencies extracts dependencies from Cargo.toml
func (a *ProjectAnalyzer) extractCargoTomlDependencies(content, source string) []Dependency {
	deps := make([]Dependency, 0)
	lines := strings.Split(content, "\n")

	inDepsBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[dependencies]" {
			inDepsBlock = true
			continue
		}

		if inDepsBlock && strings.HasPrefix(trimmed, "[") {
			inDepsBlock = false
			continue
		}

		if inDepsBlock && strings.Contains(trimmed, "=") {
			// Parse dependency line: package = "version"
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Source:  source,
				})
			}
		}
	}

	return deps
}

// AnalyzeJavaScriptFile analyzes a JavaScript/TypeScript source file and extracts code structure
func (a *ProjectAnalyzer) AnalyzeJavaScriptFile(filePath string) (*CodeStructure, error) {
	fullPath := filepath.Join(a.workspaceRoot, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JavaScript file %s: %w", filePath, err)
	}

	structure := &CodeStructure{
		Packages:   make([]PackageInfo, 0),
		Functions:  make([]FunctionInfo, 0),
		Classes:    make([]ClassInfo, 0),
		Structs:    make([]StructInfo, 0),
		Interfaces: make([]InterfaceInfo, 0),
		Exports:    make([]ExportInfo, 0),
	}

	lines := strings.Split(string(content), "\n")

	// Extract module-level comment as package description
	pkgDesc := extractJavaScriptModuleComment(lines)
	pkgInfo := PackageInfo{
		Name:        strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		Path:        filePath,
		Description: pkgDesc,
	}
	structure.Packages = append(structure.Packages, pkgInfo)

	// Extract functions, classes, and exports
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Extract class definitions
		if strings.Contains(line, "class ") {
			classInfo, nextLine := extractJavaScriptClass(lines, i, pkgInfo.Name)
			if classInfo != nil {
				structure.Classes = append(structure.Classes, *classInfo)

				// Check if exported
				if strings.Contains(line, "export") {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    classInfo.Name,
						Type:    "class",
						Package: pkgInfo.Name,
					})
				}
			}
			i = nextLine - 1
			continue
		}

		// Extract function declarations
		if strings.Contains(line, "function ") {
			funcInfo, nextLine := extractJavaScriptFunction(lines, i, pkgInfo.Name)
			if funcInfo != nil {
				structure.Functions = append(structure.Functions, *funcInfo)

				// Check if exported
				if strings.Contains(line, "export") {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    funcInfo.Name,
						Type:    "function",
						Package: pkgInfo.Name,
					})
				}
			}
			i = nextLine - 1
			continue
		}

		// Extract arrow function assignments (const/let/var name = ...)
		if (strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") || strings.HasPrefix(line, "var ")) && strings.Contains(line, "=>") {
			funcInfo := extractJavaScriptArrowFunction(line, pkgInfo.Name)
			if funcInfo != nil {
				structure.Functions = append(structure.Functions, *funcInfo)

				// Check if exported
				if strings.Contains(line, "export") {
					structure.Exports = append(structure.Exports, ExportInfo{
						Name:    funcInfo.Name,
						Type:    "function",
						Package: pkgInfo.Name,
					})
				}
			}
		}

		// Extract named exports
		if strings.HasPrefix(line, "export {") {
			exports := extractJavaScriptNamedExports(line)
			for _, exportName := range exports {
				structure.Exports = append(structure.Exports, ExportInfo{
					Name:    exportName,
					Type:    "unknown",
					Package: pkgInfo.Name,
				})
			}
		}
	}

	return structure, nil
}
