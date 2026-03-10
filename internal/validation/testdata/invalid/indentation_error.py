#!/usr/bin/env python3
"""A Python script with indentation errors."""

def bad_indentation():
    """This function has indentation errors."""
    print("Line 1")
      print("Line 2 - bad indentation")
    print("Line 3")

if __name__ == "__main__":
  bad_indentation()
