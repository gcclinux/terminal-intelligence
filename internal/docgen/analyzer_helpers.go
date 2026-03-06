package docgen

import (
	"regexp"
	"strings"
)

// Python helper functions

// extractPythonModuleDocstring extracts the module-level docstring from Python code
func extractPythonModuleDocstring(lines []string) string {
	// Look for docstring at the beginning of the file (after comments)
	for i := 0; i < len(lines) && i < 10; i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for docstring (""" or ''')
		if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, "'''") {
			delimiter := line[:3]
			docstring := strings.TrimPrefix(line, delimiter)

			// Single-line docstring
			if strings.HasSuffix(docstring, delimiter) {
				return strings.TrimSuffix(docstring, delimiter)
			}

			// Multi-line docstring
			var sb strings.Builder
			sb.WriteString(docstring)
			sb.WriteString("\n")

			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], delimiter) {
					sb.WriteString(strings.Split(lines[j], delimiter)[0])
					break
				}
				sb.WriteString(lines[j])
				sb.WriteString("\n")
			}

			return strings.TrimSpace(sb.String())
		}

		// If we hit code, stop looking
		break
	}

	return ""
}

// extractPythonClass extracts a Python class definition
func extractPythonClass(lines []string, startLine int, pkgName string) (*ClassInfo, int) {
	line := strings.TrimSpace(lines[startLine])

	// Parse class definition: class ClassName(Base):
	re := regexp.MustCompile(`^class\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, startLine + 1
	}

	className := matches[1]
	classInfo := ClassInfo{
		Name:       className,
		Package:    pkgName,
		Methods:    make([]FunctionInfo, 0),
		IsExported: !strings.HasPrefix(className, "_"),
	}

	// Extract docstring
	nextLine := startLine + 1
	if nextLine < len(lines) {
		docstring, endLine := extractPythonDocstring(lines, nextLine)
		classInfo.Comment = docstring
		nextLine = endLine
	}

	// Extract methods (functions indented within the class)
	baseIndent := getIndentLevel(lines[startLine])
	for i := nextLine; i < len(lines); i++ {
		line := lines[i]

		// Stop if we hit a line with same or less indentation (end of class)
		if strings.TrimSpace(line) != "" && getIndentLevel(line) <= baseIndent {
			return &classInfo, i
		}

		// Check for method definition
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") {
			method, endLine := extractPythonFunction(lines, i, pkgName, baseIndent+4)
			if method != nil {
				classInfo.Methods = append(classInfo.Methods, *method)
			}
			i = endLine - 1
		}
	}

	return &classInfo, len(lines)
}

// extractPythonFunction extracts a Python function definition
func extractPythonFunction(lines []string, startLine int, pkgName string, minIndent int) (*FunctionInfo, int) {
	line := strings.TrimSpace(lines[startLine])

	// Parse function definition: def function_name(params):
	re := regexp.MustCompile(`^def\s+(\w+)\s*\((.*?)\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, startLine + 1
	}

	funcName := matches[1]
	paramsStr := ""
	if len(matches) > 2 {
		paramsStr = matches[2]
	}

	funcInfo := FunctionInfo{
		Name:       funcName,
		Package:    pkgName,
		Parameters: make([]Parameter, 0),
		Returns:    make([]ReturnValue, 0),
		IsExported: !strings.HasPrefix(funcName, "_"),
	}

	// Parse parameters
	if paramsStr != "" {
		params := strings.Split(paramsStr, ",")
		for _, param := range params {
			param = strings.TrimSpace(param)
			if param == "" || param == "self" || param == "cls" {
				continue
			}

			// Handle type hints: name: type or name: type = default
			parts := strings.Split(param, ":")
			paramName := strings.TrimSpace(parts[0])
			paramType := "Any"

			if len(parts) > 1 {
				typeAndDefault := strings.TrimSpace(parts[1])
				// Remove default value if present
				if idx := strings.Index(typeAndDefault, "="); idx != -1 {
					paramType = strings.TrimSpace(typeAndDefault[:idx])
				} else {
					paramType = typeAndDefault
				}
			}

			funcInfo.Parameters = append(funcInfo.Parameters, Parameter{
				Name: paramName,
				Type: paramType,
			})
		}
	}

	// Extract return type from type hint if present
	if strings.Contains(line, "->") {
		parts := strings.Split(line, "->")
		if len(parts) > 1 {
			returnType := strings.TrimSpace(strings.TrimSuffix(parts[1], ":"))
			funcInfo.Returns = append(funcInfo.Returns, ReturnValue{
				Type: returnType,
			})
		}
	}

	// Build signature
	funcInfo.Signature = buildPythonSignature(funcInfo)

	// Extract docstring
	nextLine := startLine + 1
	if nextLine < len(lines) {
		docstring, endLine := extractPythonDocstring(lines, nextLine)
		funcInfo.Comment = docstring
		nextLine = endLine
	}

	// Find end of function (next line with same or less indentation)
	baseIndent := getIndentLevel(lines[startLine])
	for i := nextLine; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) != "" && getIndentLevel(line) <= baseIndent {
			return &funcInfo, i
		}
	}

	return &funcInfo, len(lines)
}

// extractPythonDocstring extracts a docstring starting at the given line
func extractPythonDocstring(lines []string, startLine int) (string, int) {
	if startLine >= len(lines) {
		return "", startLine
	}

	line := strings.TrimSpace(lines[startLine])

	// Check for docstring (""" or ''')
	if !strings.HasPrefix(line, `"""`) && !strings.HasPrefix(line, "'''") {
		return "", startLine
	}

	delimiter := line[:3]
	docstring := strings.TrimPrefix(line, delimiter)

	// Single-line docstring
	if strings.HasSuffix(docstring, delimiter) {
		return strings.TrimSuffix(docstring, delimiter), startLine + 1
	}

	// Multi-line docstring
	var sb strings.Builder
	sb.WriteString(docstring)
	sb.WriteString("\n")

	for i := startLine + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], delimiter) {
			sb.WriteString(strings.Split(lines[i], delimiter)[0])
			return strings.TrimSpace(sb.String()), i + 1
		}
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String()), len(lines)
}

// getIndentLevel returns the indentation level of a line
func getIndentLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

// buildPythonSignature builds a Python function signature string
func buildPythonSignature(funcInfo FunctionInfo) string {
	var sb strings.Builder
	sb.WriteString("def ")
	sb.WriteString(funcInfo.Name)
	sb.WriteString("(")

	for i, param := range funcInfo.Parameters {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.Name)
		if param.Type != "" && param.Type != "Any" {
			sb.WriteString(": ")
			sb.WriteString(param.Type)
		}
	}

	sb.WriteString(")")

	if len(funcInfo.Returns) > 0 && funcInfo.Returns[0].Type != "" {
		sb.WriteString(" -> ")
		sb.WriteString(funcInfo.Returns[0].Type)
	}

	return sb.String()
}

// JavaScript helper functions

// extractJavaScriptModuleComment extracts the module-level comment from JavaScript code
func extractJavaScriptModuleComment(lines []string) string {
	// Look for comment at the beginning of the file
	var comment strings.Builder
	inBlockComment := false

	for i := 0; i < len(lines) && i < 20; i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines
		if line == "" {
			if inBlockComment {
				comment.WriteString("\n")
			}
			continue
		}

		// Block comment start
		if strings.HasPrefix(line, "/*") {
			inBlockComment = true
			content := strings.TrimPrefix(line, "/*")
			content = strings.TrimPrefix(content, "*")
			content = strings.TrimSpace(content)

			// Check if it ends on the same line
			if strings.HasSuffix(line, "*/") {
				content = strings.TrimSuffix(content, "*/")
				content = strings.TrimSpace(content)
				comment.WriteString(content)
				return comment.String()
			}

			if content != "" {
				comment.WriteString(content)
				comment.WriteString("\n")
			}
			continue
		}

		// Inside block comment
		if inBlockComment {
			if strings.HasSuffix(line, "*/") {
				content := strings.TrimSuffix(line, "*/")
				content = strings.TrimPrefix(content, "*")
				content = strings.TrimSpace(content)
				if content != "" {
					comment.WriteString(content)
				}
				return comment.String()
			}

			content := strings.TrimPrefix(line, "*")
			content = strings.TrimSpace(content)
			if content != "" {
				comment.WriteString(content)
				comment.WriteString("\n")
			}
			continue
		}

		// Single-line comment
		if strings.HasPrefix(line, "//") {
			content := strings.TrimPrefix(line, "//")
			content = strings.TrimSpace(content)
			if content != "" {
				comment.WriteString(content)
				comment.WriteString("\n")
			}
			continue
		}

		// If we hit code, stop looking
		break
	}

	return strings.TrimSpace(comment.String())
}

// extractJavaScriptClass extracts a JavaScript class definition
func extractJavaScriptClass(lines []string, startLine int, pkgName string) (*ClassInfo, int) {
	line := strings.TrimSpace(lines[startLine])

	// Parse class definition: class ClassName or export class ClassName
	re := regexp.MustCompile(`class\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, startLine + 1
	}

	className := matches[1]
	classInfo := ClassInfo{
		Name:       className,
		Package:    pkgName,
		Methods:    make([]FunctionInfo, 0),
		IsExported: strings.Contains(line, "export"),
	}

	// Extract JSDoc comment if present (look backwards)
	if startLine > 0 {
		comment := extractJavaScriptComment(lines, startLine-1, true)
		classInfo.Comment = comment
	}

	// Find class body (between { and })
	braceCount := 0
	inClass := false

	for i := startLine; i < len(lines); i++ {
		line := lines[i]

		for _, ch := range line {
			if ch == '{' {
				braceCount++
				inClass = true
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 && inClass {
					return &classInfo, i + 1
				}
			}
		}

		// Extract methods
		if inClass && braceCount == 1 {
			trimmed := strings.TrimSpace(line)

			// Method: methodName(params) or async methodName(params)
			methodRe := regexp.MustCompile(`^\s*(async\s+)?(\w+)\s*\(`)
			if matches := methodRe.FindStringSubmatch(trimmed); len(matches) > 2 {
				methodName := matches[2]

				// Skip constructor (already implied)
				if methodName == "constructor" {
					continue
				}

				method := FunctionInfo{
					Name:       methodName,
					Package:    pkgName,
					Parameters: make([]Parameter, 0),
					Returns:    make([]ReturnValue, 0),
					IsExported: classInfo.IsExported,
				}

				// Extract parameters
				if idx := strings.Index(trimmed, "("); idx != -1 {
					if endIdx := strings.Index(trimmed[idx:], ")"); endIdx != -1 {
						paramsStr := trimmed[idx+1 : idx+endIdx]
						method.Parameters = parseJavaScriptParameters(paramsStr)
					}
				}

				method.Signature = buildJavaScriptSignature(method)
				classInfo.Methods = append(classInfo.Methods, method)
			}
		}
	}

	return &classInfo, len(lines)
}

// extractJavaScriptFunction extracts a JavaScript function declaration
func extractJavaScriptFunction(lines []string, startLine int, pkgName string) (*FunctionInfo, int) {
	line := strings.TrimSpace(lines[startLine])

	// Parse function declaration: function functionName(params) or export function functionName(params)
	re := regexp.MustCompile(`function\s+(\w+)\s*\((.*?)\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, startLine + 1
	}

	funcName := matches[1]
	paramsStr := ""
	if len(matches) > 2 {
		paramsStr = matches[2]
	}

	funcInfo := FunctionInfo{
		Name:       funcName,
		Package:    pkgName,
		Parameters: parseJavaScriptParameters(paramsStr),
		Returns:    make([]ReturnValue, 0),
		IsExported: strings.Contains(line, "export"),
	}

	// Extract JSDoc comment if present (look backwards)
	if startLine > 0 {
		comment := extractJavaScriptComment(lines, startLine-1, true)
		funcInfo.Comment = comment
	}

	funcInfo.Signature = buildJavaScriptSignature(funcInfo)

	// Find end of function (matching braces)
	braceCount := 0
	for i := startLine; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					return &funcInfo, i + 1
				}
			}
		}
	}

	return &funcInfo, len(lines)
}

// extractJavaScriptArrowFunction extracts an arrow function from a line
func extractJavaScriptArrowFunction(line string, pkgName string) *FunctionInfo {
	// Parse: const/let/var name = (params) => or const/let/var name = params =>
	re := regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(?(.*?)\)?\s*=>`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}

	funcName := matches[1]
	paramsStr := ""
	if len(matches) > 2 {
		paramsStr = matches[2]
	}

	funcInfo := FunctionInfo{
		Name:       funcName,
		Package:    pkgName,
		Parameters: parseJavaScriptParameters(paramsStr),
		Returns:    make([]ReturnValue, 0),
		IsExported: strings.Contains(line, "export"),
	}

	funcInfo.Signature = buildJavaScriptSignature(funcInfo)

	return &funcInfo
}

// extractJavaScriptNamedExports extracts named exports from export { ... } statement
func extractJavaScriptNamedExports(line string) []string {
	// Parse: export { name1, name2, name3 }
	re := regexp.MustCompile(`export\s*\{\s*([^}]+)\s*\}`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}

	exportsStr := matches[1]
	exports := strings.Split(exportsStr, ",")
	result := make([]string, 0, len(exports))

	for _, exp := range exports {
		exp = strings.TrimSpace(exp)
		// Handle "name as alias" - we want the original name
		if idx := strings.Index(exp, " as "); idx != -1 {
			exp = strings.TrimSpace(exp[:idx])
		}
		if exp != "" {
			result = append(result, exp)
		}
	}

	return result
}

// extractJavaScriptComment extracts a comment before a declaration
func extractJavaScriptComment(lines []string, startLine int, lookBackwards bool) string {
	if startLine < 0 || startLine >= len(lines) {
		return ""
	}

	var comment strings.Builder

	if lookBackwards {
		// Look backwards for JSDoc or single-line comments
		for i := startLine; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])

			// Empty line - stop looking
			if line == "" {
				break
			}

			// JSDoc comment end
			if strings.HasSuffix(line, "*/") {
				// Find the start of the comment
				for j := i; j >= 0; j-- {
					commentLine := strings.TrimSpace(lines[j])
					if strings.HasPrefix(commentLine, "/**") || strings.HasPrefix(commentLine, "/*") {
						// Extract comment content
						for k := j; k <= i; k++ {
							content := strings.TrimSpace(lines[k])
							content = strings.TrimPrefix(content, "/**")
							content = strings.TrimPrefix(content, "/*")
							content = strings.TrimSuffix(content, "*/")
							content = strings.TrimPrefix(content, "*")
							content = strings.TrimSpace(content)
							if content != "" {
								if comment.Len() > 0 {
									comment.WriteString("\n")
								}
								comment.WriteString(content)
							}
						}
						return comment.String()
					}
				}
				break
			}

			// Single-line comment
			if strings.HasPrefix(line, "//") {
				content := strings.TrimPrefix(line, "//")
				content = strings.TrimSpace(content)
				if content != "" {
					if comment.Len() > 0 {
						comment.WriteString("\n")
					}
					comment.WriteString(content)
				}
				continue
			}

			// Not a comment - stop looking
			break
		}
	}

	return comment.String()
}

// parseJavaScriptParameters parses JavaScript function parameters
func parseJavaScriptParameters(paramsStr string) []Parameter {
	if paramsStr == "" {
		return make([]Parameter, 0)
	}

	params := strings.Split(paramsStr, ",")
	result := make([]Parameter, 0, len(params))

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		// Handle destructuring, default values, etc. - just get the name
		paramName := param
		paramType := "any"

		// Remove default value
		if idx := strings.Index(param, "="); idx != -1 {
			paramName = strings.TrimSpace(param[:idx])
		}

		// Handle TypeScript type annotation
		if idx := strings.Index(paramName, ":"); idx != -1 {
			parts := strings.Split(paramName, ":")
			paramName = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				paramType = strings.TrimSpace(parts[1])
			}
		}

		// Handle rest parameters
		paramName = strings.TrimPrefix(paramName, "...")

		result = append(result, Parameter{
			Name: paramName,
			Type: paramType,
		})
	}

	return result
}

// buildJavaScriptSignature builds a JavaScript function signature string
func buildJavaScriptSignature(funcInfo FunctionInfo) string {
	var sb strings.Builder
	sb.WriteString("function ")
	sb.WriteString(funcInfo.Name)
	sb.WriteString("(")

	for i, param := range funcInfo.Parameters {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.Name)
	}

	sb.WriteString(")")

	return sb.String()
}
