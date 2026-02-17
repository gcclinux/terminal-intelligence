# Property-Based Tests

This directory contains property-based tests using gopter.

Each property test validates universal properties that should hold across all inputs.

## Running Property Tests

```bash
go test ./tests/property/...
```

## Test Format

Each property test should:
- Reference the design document property number
- Use minimum 100 iterations
- Include tag: `// Feature: Terminal Intelligence (TI), Property {number}: {property_text}`
