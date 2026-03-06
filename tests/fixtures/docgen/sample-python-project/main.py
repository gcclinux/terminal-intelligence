"""Sample Python module for testing documentation generation."""

def hello_world():
    """Print a greeting message."""
    print("Hello, World!")

class Greeter:
    """A class for greeting people."""
    
    def __init__(self, name):
        """Initialize the greeter with a name."""
        self.name = name
    
    def greet(self):
        """Print a personalized greeting."""
        print(f"Hello, {self.name}!")

if __name__ == "__main__":
    hello_world()
