# Config Editor - Interactive Demo

## Visual Walkthrough

This document shows a complete editing session with the config editor, demonstrating the cursor navigation features.

---

## Scenario: Switching from Ollama to Gemini

### Step 1: Open Config Editor

```
┌────────────────────────────────────────────────────────────┐
│                     AI Chat Pane                           │
│                                                            │
│  TI> /config█                                              │
│                                                            │
│                                                            │
└────────────────────────────────────────────────────────────┘

User presses Enter...
```

### Step 2: Config Editor Opens

```
┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  > agent: ollama                    ← Selected             │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 3: Edit Agent Field

```
User presses Enter to edit...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  > agent: ollama█                   ← Editing, cursor at end│
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 4: Position Cursor at Start

```
User presses Home...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  > agent: █ollama                   ← Cursor at start      │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 5: Delete "ollama"

```
User presses Delete 6 times...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  > agent: █                         ← Empty field          │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 6: Type "gemini"

```
User types 'gemini'...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  > agent: gemini█                   ← New value            │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 7: Save and Move to Next Field

```
User presses Enter to save, then ↓ to move down...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: gemini                    ← Saved                │
│  > model: llama2                    ← Now selected         │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 8: Skip to gemini_api Field

```
User presses ↓ three times...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: gemini                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│  > gemini_api:                      ← Selected             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 9: Add API Key

```
User presses Enter and types API key...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: gemini                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│    ollama_url: http://localhost:11434                      │
│  > gemini_api: AIzaSyD...xyz123█    ← Typing API key      │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Step 10: Save and Exit

```
User presses Enter to save, then Esc to exit...

┌────────────────────────────────────────────────────────────┐
│                     AI Chat Pane                           │
│                                                            │
│  TI> █                                                     │
│                                                            │
│                                                            │
└────────────────────────────────────────────────────────────┘

Status Bar: Configuration saved successfully to ~/.ti/config.json
```

---

## Scenario 2: Fixing a Typo

### Initial State

```
┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhots:11434  ← Typo: "hots"     │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Press Enter to Edit

```
┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhots:11434█                     │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Navigate to Typo

```
User presses ← repeatedly to position cursor...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhots█:11434  ← Cursor at 's'   │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Delete the 's'

```
User presses Backspace...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhot█:11434  ← 's' deleted      │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Type 'st'

```
User types 'st'...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhost█:11434  ← Fixed!          │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Save

```
User presses Enter...

┌────────────────────────────────────────────────────────────┐
│              Configuration Editor                          │
├────────────────────────────────────────────────────────────┤
│  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor |   │
│  [Esc] Save & Exit                                         │
├────────────────────────────────────────────────────────────┤
│                                                            │
│    agent: ollama                                           │
│    model: llama2                                           │
│    gmodel: gemini-2.5-flash-lite                           │
│  > ollama_url: http://localhost:11434  ← Saved            │
│    gemini_api:                                             │
│    workspace: /home/user/ti-workspace                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

---

## Key Takeaways

1. **Cursor shows position**: The █ block always indicates where you are
2. **Arrow keys work intuitively**: ← and → move character by character
3. **Home/End for quick jumps**: No need to hold arrow keys
4. **Insert anywhere**: Characters insert at cursor, not just at end
5. **Delete works both ways**: Backspace before cursor, Delete at cursor
6. **Visual feedback**: You always know where you are and what you're editing

This makes editing configuration values as easy as using any modern text editor!
