# Whatsabladerunner Design

## TL;DR
It is a bot that you command via WhatsApp self-chat, allowing the bot to speak on your behalf in other conversations.

## Code

The architecture is simple: event-based with concurrency, aiming to apply idiomatic Go practices.

There are only two key parts:

### 1. Integration with `whatsmeow` for WhatsApp communications

This component follows events from `whatsmeow` and forwards required information to the bot core.

### 2. Bot Core

The bot core applies pre-set prompt workflows to the received events and forwards the responses to `whatsmeow`.

Each step of the state machine representing a workflow will be an event to allow extensibility and easy testing. Event trees should be documented.

## Governance

Whatsabladerunner (referred to as "Blady" from now on) is governed by the user using three key elements:

1. Tasks
2. Behaviors
3. Watchers

### 1. Tasks

The user asks Blady to talk to a contact or group with a specific goal.

Blady will create a task and confirm it with the user.

Once a task is confirmed, Blady will gather information (read the conversation log, ask for missing information, etc.) and start chatting.

After every response, Blady will evaluate and document task progress, continuing to chat until the task is finished or blocked.

### 2. Behaviors

The user programs Blady with behaviors, which can then be enabled in specific conversations.

Once a behavior is enabled, Blady will evaluate every message in the conversation against that behavior until it is disabled.

### 3. Watchers

Blady is not responsible for its actions; it is "just following orders."

Watchers will inspect every action Blady attempts to execute (including sending messages) and evaluate if it should be allowed to proceed.

If a watcher decides to block an action, it will inform the user of the blocked action and the reason.

The user can decide to:

1. Accept the block and stop Blady from proceeding.
2. Ask Blady to try again.
3. Overrule the watcher and allow the action to proceed.

Exceptionally, a task can be programmed to always do **1** or **2**, but not **3** (while this is possible, it is **highly** discouraged).

## Config and Execution

The configuration (tasks, behaviors, watchers) will be stored in plain text in a structured directory.

While new tasks and behaviors can be created by the user talking to Blady, watchers can only be modified by the user directly editing the files.

Whatsabladerunner will run locally from the terminal and, by default, will use local LLMs and SQLite storage.

The user can configure remote LLM providers.