#!/usr/bin/env python3
"""A simple valid Python script."""

def greet(name):
    """Greet someone by name."""
    return f"Hello, {name}!"

def add(a, b):
    """Add two numbers."""
    return a + b

if __name__ == "__main__":
    print(greet("World"))
    print(f"2 + 3 = {add(2, 3)}")
