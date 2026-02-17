import psutil
import time
import os

def get_size(bytes, suffix="B"):
    """
    Scale bytes to its proper format
    e.g:
        1253656 => '1.20MB'
        1253656678 => '1.17GB'
    """
    factor = 1024
    for unit in ["", "K", "M", "G", "T", "P"]:
        if bytes < factor:
            return f"{bytes:.2f}{unit}{suffix}"
        bytes /= factor

def clear_screen():
    # Clear screen using 'cls' on Windows or 'clear' on Linux/Mac
    os.system('cls' if os.name == 'nt' else 'clear')

try:
    while True:
        print("="*40)
        print(f"  SYSTEM MONITORING - {time.strftime('%Y-%m-%d %H:%M:%S')}")  # Move date and time to the top
        
        # Get CPU usage
        cpu_usage = psutil.cpu_percent(interval=1)
        
        # Get Memory usage
        memory = psutil.virtual_memory()
        
        clear_screen()
        
        print(f"CPU Usage:      {cpu_usage}%")
        
        print(f"Memory Usage:   {memory.percent}%")
        print(f"Memory Used:    {get_size(memory.used)}")
        print(f"Memory Total:   {get_size(memory.total)}")
        print("-" * 40)
        print("Press Ctrl+C to exit")

except KeyboardInterrupt:
    print("\nMonitor stopped.")
