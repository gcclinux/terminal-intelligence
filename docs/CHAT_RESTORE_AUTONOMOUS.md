# Chat Restore with Autonomous State

## Overview

This document describes how TI restores autonomous creation state when loading a saved chat session.

## Problem

When a user:
1. Runs `/create <description>` to start autonomous app creation
2. AI generates a plan and asks "Do you want to proceed? Type /proceed to continue or /cancel to abort."
3. User exits TI before responding
4. User restarts TI and loads the chat with Ctrl+L
5. User types `/proceed`

Previously, this would fail with "Nothing to proceed with" because the internal `AutonomousCreator` state was not persisted or restored.

## Solution

When loading a chat session with Ctrl+L, TI now:

1. Parses the loaded chat messages
2. Detects if the last assistant message contains the proceed prompt
3. Extracts the plan and original `/create` description
4. Reconstructs the `AutonomousCreator` in `StateWaitingApproval`
5. Allows the user to continue with `/proceed`

## Implementation Details

### Key Functions

- `loadChatHistory()`: Parses saved chat file and restores messages
- `restoreAutonomousState()`: Detects and restores autonomous creation state
- `extractProjectNameFromPlan()`: Extracts project name from plan text

### State Restoration

The `AutonomousCreator` is reconstructed with:
- `AIClient`: Current AI client
- `Model`: Current model from config
- `Workspace`: Current workspace directory
- `Description`: Extracted from original `/create` command
- `Plan`: Extracted from assistant's response
- `ProjectName`: Extracted from plan text
- `ProjectDir`: Computed from workspace + project name
- `State`: Set to `StateWaitingApproval`

### Detection Logic

A chat is considered to have a pending autonomous creation if:
1. Last message is from assistant
2. Message contains "Do you want to proceed? Type /proceed to continue or /cancel to abort."
3. Message contains "Plan generated:"
4. A previous user message contains `/create`

## Usage

1. Start autonomous creation: `/create A simple web server`
2. AI generates plan and waits for approval
3. Exit TI (Ctrl+Q)
4. Restart TI in the same workspace
5. Load chat: Ctrl+L, select the session
6. Continue: `/proceed`

The autonomous creation will resume from where it left off.

## Limitations

- Only works for autonomous creation (`/create` command)
- Does not restore state for project-wide fixes (`/project` command)
- Requires the original `/create` command to be in the chat history
- Project name extraction is best-effort and may fall back to default

## Future Enhancements

- Persist and restore `/project` preview state
- Save state to a separate metadata file for more reliable restoration
- Support resuming from any autonomous creation state, not just waiting for approval
