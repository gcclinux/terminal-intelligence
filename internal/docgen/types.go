package docgen

// DocumentationType represents the type of documentation to generate
type DocumentationType int

const (
	// DocTypeUserManual represents user manual documentation
	DocTypeUserManual DocumentationType = iota
	// DocTypeInstallation represents installation guide documentation
	DocTypeInstallation
	// DocTypeAPI represents API reference documentation
	DocTypeAPI
	// DocTypeTutorial represents tutorial documentation
	DocTypeTutorial
	// DocTypeGeneral represents general documentation
	DocTypeGeneral
)

// String returns the string representation of the documentation type
func (d DocumentationType) String() string {
	switch d {
	case DocTypeUserManual:
		return "User Manual"
	case DocTypeInstallation:
		return "Installation Guide"
	case DocTypeAPI:
		return "API Reference"
	case DocTypeTutorial:
		return "Tutorial"
	case DocTypeGeneral:
		return "General Documentation"
	default:
		return "Unknown"
	}
}

// Filename returns the standard filename for the documentation type
func (d DocumentationType) Filename() string {
	switch d {
	case DocTypeUserManual:
		return "USER_MANUAL.md"
	case DocTypeInstallation:
		return "INSTALLATION.md"
	case DocTypeAPI:
		return "API_REFERENCE.md"
	case DocTypeTutorial:
		return "TUTORIAL.md"
	case DocTypeGeneral:
		return "DOCUMENTATION.md"
	default:
		return "DOCUMENTATION.md"
	}
}

// ParsedCommand represents a parsed user command with documentation flags
type ParsedCommand struct {
	IsProjectWide   bool     // /project flag present
	IsDocRequest    bool     // /doc flag present
	NaturalLanguage string   // Remaining text after flags
	ScopeFilters    []string // File patterns or module names
}

// ClassificationResult represents the result of classifying a documentation request
type ClassificationResult struct {
	Types      []DocumentationType // Identified doc types
	Confidence float64             // Match confidence (0.0-1.0)
}

// Parameter represents a function parameter
type Parameter struct {
	Name string
	Type string
}

// ReturnValue represents a function return value
type ReturnValue struct {
	Name string
	Type string
}

// PackageInfo represents information about a package
type PackageInfo struct {
	Name        string
	Path        string
	Description string
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string
	Package    string
	Signature  string
	Parameters []Parameter
	Returns    []ReturnValue
	Comment    string
	IsExported bool
}

// ClassInfo represents information about a class/struct
type ClassInfo struct {
	Name       string
	Package    string
	Methods    []FunctionInfo
	Comment    string
	IsExported bool
}

// StructInfo represents information about a Go struct
type StructInfo struct {
	Name       string
	Package    string
	Fields     []FieldInfo
	Methods    []FunctionInfo
	Comment    string
	IsExported bool
}

// FieldInfo represents a struct field
type FieldInfo struct {
	Name    string
	Type    string
	Tag     string
	Comment string
}

// InterfaceInfo represents information about an interface
type InterfaceInfo struct {
	Name       string
	Package    string
	Methods    []MethodSignature
	Comment    string
	IsExported bool
}

// MethodSignature represents an interface method signature
type MethodSignature struct {
	Name       string
	Parameters []Parameter
	Returns    []ReturnValue
}

// ExportInfo represents information about an exported symbol
type ExportInfo struct {
	Name    string
	Type    string // "function", "class", "constant", "type"
	Package string
}

// CodeStructure represents the structure of code in the project
type CodeStructure struct {
	Packages   []PackageInfo
	Functions  []FunctionInfo
	Classes    []ClassInfo
	Structs    []StructInfo
	Interfaces []InterfaceInfo
	Exports    []ExportInfo
}

// ConfigInfo represents configuration information from the project
type ConfigInfo struct {
	PackageManifests []string // go.mod, package.json, requirements.txt
	BuildScripts     []string // Makefile, build scripts
	ConfigFiles      []string // .env, config files
}

// CommentBlock represents a block of comments in code
type CommentBlock struct {
	File    string
	Line    int
	Content string
}

// DocstringBlock represents a docstring for a function
type DocstringBlock struct {
	Function string
	Content  string
}

// ExistingDocs represents existing documentation in the project
type ExistingDocs struct {
	ReadmeContent string
	Comments      []CommentBlock
	Docstrings    []DocstringBlock
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string
	Version string
	Source  string // "go.mod", "package.json", etc.
}

// DependencyInfo represents dependency information
type DependencyInfo struct {
	Runtime []Dependency
	Build   []Dependency
}

// AnalysisResult represents the result of analyzing a project
type AnalysisResult struct {
	CodeStructure *CodeStructure
	Configuration *ConfigInfo
	Documentation *ExistingDocs
	Dependencies  *DependencyInfo
}

// GeneratedDoc represents a generated documentation file
type GeneratedDoc struct {
	Type     DocumentationType // Type of documentation
	Content  string            // Markdown formatted content
	Filename string            // Target filename
}

// WriteResult represents the result of writing a documentation file
type WriteResult struct {
	Filename string // Name of file written
	Path     string // Full path to file
	Existed  bool   // Whether file existed before
	Written  bool   // Whether write succeeded
}
