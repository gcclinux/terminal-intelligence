# Manual Testing Guide for Automatic Code Validation

This document provides instructions for manually testing the automatic code validation system.

## Test Fixtures Overview

The `testdata` directory contains sample projects for manual testing:

- `valid/` - Contains valid code files that should compile/validate successfully
- `invalid/` - Contains code files with intentional errors for testing error handling
- `rapid/` - Contains test files used by property-based tests

## Test Scenarios

### Scenario 1: Valid Go Code Validation

**Files**: `testdata/valid/main.go`, `testdata/valid/utils.go`

**Expected Behavior**:
1. System detects Go files
2. Executes `go build` for the package
3. Displays success message: "✅ Go compilation successful (X.Xs)"

**Manual Test**:
```bash
cd internal/validation/testdata/valid
go build .
# Should compile successfully
```

### Scenario 2: Invalid Go Code Validation

**Files**: `testdata/invalid/syntax_error.go`, `testdata/invalid/undefined_var.go`

**Expected Behavior**:
1. System detects Go files
2. Executes `go build` for the package
3. Displays failure message with error details:
   - File path
   - Line number
   - Column number
   - Error message

**Manual Test**:
```bash
cd internal/validation/testdata/invalid
go build .
# Should fail with compilation errors
```

### Scenario 3: Valid Python Code Validation

**Files**: `testdata/valid/script.py`, `testdata/valid/module.py`

**Expected Behavior**:
1. System detects Python files
2. Executes `python -m py_compile` for each file independently
3. Displays success message for each file

**Manual Test**:
```bash
cd internal/validation/testdata/valid
python -m py_compile script.py
python -m py_compile module.py
# Should validate successfully
```

### Scenario 4: Invalid Python Code Validation

**Files**: `testdata/invalid/syntax_error.py`, `testdata/invalid/indentation_error.py`

**Expected Behavior**:
1. System detects Python files
2. Executes `python -m py_compile` for each file
3. Displays failure message with error details:
   - File path
   - Line number
   - Error message

**Manual Test**:
```bash
cd internal/validation/testdata/invalid
python -m py_compile syntax_error.py
python -m py_compile indentation_error.py
# Should fail with syntax/indentation errors
```

### Scenario 5: Mixed Language Validation

**Files**: Mix of Go and Python files

**Expected Behavior**:
1. System groups files by language
2. Validates Go files as a package
3. Validates Python files independently
4. Displays results for each language separately

### Scenario 6: Unsupported File Types

**Files**: `.txt`, `.md`, `.json` files

**Expected Behavior**:
1. System detects unsupported file types
2. Displays notification: "ℹ️ Skipped validation for unsupported files"
3. Lists skipped files
4. Lists supported languages: "Go, Python"

### Scenario 7: Long-Running Validation

**Files**: Large Go project or many Python files

**Expected Behavior**:
1. System starts validation
2. After 5 seconds, displays progress indicator: "⏳ Compiling Go package... (X.Xs)"
3. Updates progress message every second
4. Displays final result when complete

### Scenario 8: Configuration System

**Test**: Add custom language configuration

**Expected Behavior**:
1. Create LanguageConfig for new language (e.g., JavaScript)
2. Register configuration with LanguageDetector
3. System detects files with configured extensions
4. Executes configured validator command

**Example**:
```go
config := LanguageConfig{
    Name:       "JavaScript",
    Extensions: []string{".js", ".jsx"},
    Validator: ValidatorConfig{
        Command: "eslint",
        Args:    []string{"--format", "json"},
        Timeout: 30 * time.Second,
    },
}
detector.RegisterLanguage(config)
```

## Integration Testing

### End-to-End Flow

1. **Setup**: Create a test workspace with mixed language files
2. **Trigger**: Simulate AI file modification event
3. **Verify**:
   - File change detector captures events
   - Language detector identifies languages correctly
   - Validation engine groups files by language
   - Compiler interface executes appropriate validators
   - Chat panel displays formatted results

### Error Handling

Test the following error conditions:

1. **Timeout**: Validation exceeds configured timeout
   - Expected: "⚠️ Validation timed out after Xs"

2. **Command Not Found**: Validator command not in PATH
   - Expected: "❌ Validator not found: {command}. Please ensure {language} is installed."

3. **Permission Denied**: Cannot read/execute files
   - Expected: "❌ Permission denied: {file}"

4. **File Not Found**: File deleted after change event
   - Expected: "⚠️ File not found: {file} (may have been deleted)"

5. **Unparseable Output**: Compiler output doesn't match expected format
   - Expected: Raw output displayed with note: "⚠️ Could not parse error details"

## Performance Testing

### Validation Speed

Measure validation time for different scenarios:

1. Single Go file: < 1 second
2. Go package with 10 files: < 3 seconds
3. Single Python file: < 0.5 seconds
4. 10 Python files: < 2 seconds

### Concurrent Validation

Test multiple validation requests:

1. Trigger validation for Go files
2. While running, trigger validation for Python files
3. Verify: Second validation is queued and executes after first completes

## Coverage Verification

Run tests with coverage:

```bash
go test -cover ./internal/validation/...
```

Expected coverage: > 90%

## Property-Based Test Verification

All 26 correctness properties should pass:

1. Property 1: File Change Event Capture
2. Property 2: Multiple File Event Capture
3. Property 3: Language Detection from Extension
4. Property 4: Configured Language Detection
5. Property 5: Unsupported Language Reporting
6. Property 6: Complete Output Capture
7. Property 7: Success/Failure Determination
8. Property 8: Validation Result Production
9. Property 9: Go Package Compilation
10. Property 10: Success Result Consistency
11. Property 11: Error Message Inclusion
12. Property 12: Error Location Parsing
13. Property 13: Python Syntax Error Detection
14. Property 14: Python Independent Validation
15. Property 15: Validation Start Message
16. Property 16: Validation Progress Display
17. Property 17: Success Message Display
18. Property 18: Error Message Display
19. Property 19: Error Suggestion Preservation
20. Property 20: Original Error Format Preservation
21. Property 21: Language-Based File Grouping
22. Property 22: Per-Unit Status Display
23. Property 23: Unsupported File Notification
24. Property 24: Unsupported File Non-Blocking
25. Property 25: Supported Languages Listing
26. Property 26: Validation Timing Round-Trip

Run property tests:

```bash
go test -v -run "TestProperty" ./internal/validation/...
```

## Troubleshooting

### Tests Failing

1. Ensure Go and Python are installed and in PATH
2. Check file permissions in testdata directory
3. Verify test fixtures are not corrupted
4. Run tests with `-v` flag for verbose output

### Validation Not Triggering

1. Verify file change detector is properly hooked into AI file operations
2. Check that file extensions are recognized
3. Ensure validators are registered in compiler interface

### Incorrect Error Parsing

1. Verify error pattern regex matches compiler output format
2. Check that error output is captured correctly (stdout and stderr)
3. Test with actual compiler output samples

## Conclusion

This manual testing guide covers all major scenarios for the automatic code validation system. Follow these tests to verify the implementation meets all requirements and handles edge cases correctly.
