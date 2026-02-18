# Config Editor Flow Diagram

## User Interaction Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     TI Application                          │
│  ┌────────────────┐              ┌────────────────────────┐ │
│  │                │              │                        │ │
│  │  Editor Pane   │              │    AI Chat Pane        │ │
│  │                │              │                        │ │
│  │  [Code here]   │              │  TI> /config           │ │
│  │                │              │                        │ │
│  └────────────────┘              └────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                          ↓
                    User types /config
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                Configuration Editor                         │
│  ┌────────────────┐              ┌────────────────────────┐ │
│  │                │              │ Configuration Editor   │ │
│  │  Editor Pane   │              │ ─────────────────────  │ │
│  │                │              │ [↑↓] Navigate          │ │
│  │  [Code here]   │              │ [Enter] Edit/Save      │ │
│  │                │              │ [Esc] Save & Exit      │ │
│  │                │              │                        │ │
│  │                │              │ > agent: ollama        │ │
│  │                │              │   model: llama2        │ │
│  │                │              │   gmodel: gemini-...   │ │
│  │                │              │   ollama_url: http:... │ │
│  │                │              │   gemini_api:          │ │
│  │                │              │   workspace: /home/... │ │
│  └────────────────┘              └────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                          ↓
                  User navigates with ↑↓
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                    Editing a Field                          │
│  ┌────────────────┐              ┌────────────────────────┐ │
│  │                │              │ Configuration Editor   │ │
│  │  Editor Pane   │              │ ─────────────────────  │ │
│  │                │              │ [↑↓] Navigate          │ │
│  │  [Code here]   │              │ [Enter] Edit/Save      │ │
│  │                │              │ [Esc] Save & Exit      │ │
│  │                │              │                        │ │
│  │                │              │   agent: ollama        │ │
│  │                │              │ > model: qwen2.5█      │ │
│  │                │              │   gmodel: gemini-...   │ │
│  │                │              │   ollama_url: http:... │ │
│  │                │              │   gemini_api:          │ │
│  │                │              │   workspace: /home/... │ │
│  └────────────────┘              └────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                          ↓
                  User presses Enter to save
                          ↓
                  User presses Esc to exit
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                   Config Saved & Applied                    │
│  ┌────────────────┐              ┌────────────────────────┐ │
│  │                │              │                        │ │
│  │  Editor Pane   │              │    AI Chat Pane        │ │
│  │                │              │                        │ │
│  │  [Code here]   │              │  TI>                   │ │
│  │                │              │                        │ │
│  └────────────────┘              └────────────────────────┘ │
│                                                             │
│  Status: Configuration saved successfully to ~/.ti/config.json │
└─────────────────────────────────────────────────────────────┘
```

## State Transitions

```
┌──────────────┐
│ Normal Mode  │
│ (Chat Input) │
└──────┬───────┘
       │
       │ User types /config
       ↓
┌──────────────┐
│ Config Mode  │
│ (View Fields)│
└──────┬───────┘
       │
       │ User presses Enter on field
       ↓
┌──────────────┐
│ Edit Mode    │
│ (Edit Value) │
└──────┬───────┘
       │
       │ User presses Enter
       ↓
┌──────────────┐
│ Config Mode  │
│ (View Fields)│
└──────┬───────┘
       │
       │ User presses Esc
       ↓
┌──────────────┐
│ Save Config  │
│ (Validate &  │
│  Write File) │
└──────┬───────┘
       │
       │ Success
       ↓
┌──────────────┐
│ Normal Mode  │
│ (Chat Input) │
└──────────────┘
```

## Component Interaction

```
┌─────────────────────────────────────────────────────────────┐
│                         User Input                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ "/config"
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    App.handleAIMessage()                    │
│  • Detects /config command                                  │
│  • Calls config.ConfigFilePath()                            │
│  • Calls config.LoadFromFile()                              │
│  • Prepares fields and values arrays                        │
│  • Calls aiPane.EnterConfigMode()                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│              AIChatPane.EnterConfigMode()                   │
│  • Sets configMode = true                                   │
│  • Stores fields and values                                 │
│  • Initializes selectedField = 0                            │
│  • Sets editingField = false                                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│               AIChatPane.renderConfigMode()                 │
│  • Renders title bar                                        │
│  • Renders instructions                                     │
│  • Renders field list with highlighting                     │
│  • Shows cursor when editing                                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ User navigates and edits
                         ↓
┌─────────────────────────────────────────────────────────────┐
│              AIChatPane.handleKeyPress()                    │
│  • Handles navigation (↑↓)                                  │
│  • Handles edit mode (Enter)                                │
│  • Handles character input                                  │
│  • Handles save (Esc) → returns SaveConfigMsg              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ SaveConfigMsg
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                     App.Update()                            │
│  • Receives SaveConfigMsg                                   │
│  • Builds JSONConfig from fields/values                     │
│  • Calls config.Validate()                                  │
│  • Calls config.ToJSON()                                    │
│  • Writes to file with os.WriteFile()                       │
│  • Calls config.ApplyToAppConfig()                          │
│  • Reinitializes AI client                                  │
│  • Updates status message                                   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    Config File Updated                      │
│                  ~/.ti/config.json                          │
│  {                                                          │
│    "agent": "ollama",                                       │
│    "model": "qwen2.5-coder:3b",                             │
│    "gmodel": "gemini-3-pro-preview",                        │
│    "ollama_url": "http://localhost:11434",                  │
│    "gemini_api": "",                                        │
│    "workspace": "/home/user/ti-workspace"                   │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

## Keyboard Event Handling

```
Config Mode (Not Editing)
─────────────────────────
  ↑ or k  → selectedField--
  ↓ or j  → selectedField++
  Enter   → Start editing (editingField = true, editBuffer = current value)
  Esc/q   → Exit config mode (send SaveConfigMsg)

Config Mode (Editing)
────────────────────
  Enter     → Save value (configValues[selectedField] = editBuffer)
  Esc       → Cancel edit (editingField = false)
  Backspace → Delete character from editBuffer
  [char]    → Add character to editBuffer
```

## Data Flow

```
┌──────────────┐
│ config.json  │
│    File      │
└──────┬───────┘
       │
       │ LoadFromFile()
       ↓
┌──────────────┐
│  JSONConfig  │
│   Struct     │
└──────┬───────┘
       │
       │ Extract fields/values
       ↓
┌──────────────┐
│ Config Mode  │
│ UI Display   │
└──────┬───────┘
       │
       │ User edits
       ↓
┌──────────────┐
│ Modified     │
│ Values Array │
└──────┬───────┘
       │
       │ Build JSONConfig
       ↓
┌──────────────┐
│  JSONConfig  │
│   Struct     │
└──────┬───────┘
       │
       │ Validate()
       ↓
┌──────────────┐
│  Validated   │
│  Config      │
└──────┬───────┘
       │
       │ ToJSON()
       ↓
┌──────────────┐
│ Pretty JSON  │
│   Bytes      │
└──────┬───────┘
       │
       │ WriteFile()
       ↓
┌──────────────┐
│ config.json  │
│    File      │
└──────────────┘
       │
       │ ApplyToAppConfig()
       ↓
┌──────────────┐
│  AppConfig   │
│   Updated    │
└──────────────┘
```
