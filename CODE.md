# whatsabladerunner: Codebase Overview

This document provides a high-level overview of the `whatsabladerunner` codebase, its package structure, and core components to help navigate the project.

## Project Structure

The project is structured as a Go application with various internal packages located in the `pkg/` directory.

### Root Area

- `main.go`: The **orchestration hub**. It initializes all components, handles WhatsApp events (from `whatsmeow`), and routes messages to either the setup flow (`pkg/batata`), the watcher logic, or the bot processing (`pkg/bot`).

---

### Core Packages (`pkg/`)

#### [`pkg/batata`](./pkg/batata)

**The Communication & Configuration Hub.**

- **Purpose**: Manages the initial setup flow, language selection, and LLM configuration via WhatsApp messages. It's critical for user onboarding and system configuration.
- **Main Types**: `Kernel` (state manager), `Config` (persistent settings).
- **Core Logic**: `HandleMessage` processes step-by-step setup interactions.

#### [`pkg/bot`](./pkg/bot)

**The Intelligence Layer.**

- **Purpose**: Processes messages using LLMs and translates intent into executable actions.
- **Main Types**: `Bot` (main processor), `BotResponse` (processed LLM output).
- **Core Logic**: `Process` (general chat) and `ProcessTask` (focused agent work).
- **Security**: Prompts in `config/modes/` incorporate injection guards and bot-suspicion awareness to maintain persona integrity.

#### [`pkg/bot/actions`](./pkg/bot/actions)

**The Extensibility Layer.**

- **Purpose**: Defines and registers actions the LLM can perform (e.g., `memory_update`, `create_task`, `send_media`, `enable_behavior`).
- **Main Types**: `Registry` (action storage), `Action` (interface for implementations).
- **Integration**: New functionality should be added as an `Action` and registered in `bot.go`.

#### [`pkg/tasks`](./pkg/tasks)

**The Persistence Layer for Work.**

- **Purpose**: Manages long-running tasks, their lifecycle (Pending, Running, Finished), and scheduling.
- **Main Types**: `TaskManager` (file-based CRUD), `Task` (structure representing work).
- **Integrations**: `CheckScheduledTasks` is called periodically by a ticker in `main.go`.

#### [`pkg/behaviors`](./pkg/behaviors)

**The Persona/Rules Layer.**

- **Purpose**: Manages active behavioral directives (personas/rules) enabled for specific contacts.
- **Main Types**: `BehaviorManager`, `Behavior`.
- **Core Logic**: `ProcessBehaviors` in `bot.go` injects behavior content into the prompt.

#### [`pkg/llm`](./pkg/llm)

**The LLM Client Interface.**

- **Purpose**: Provides a unified interface for different LLM providers.
- **Main Types**: `Client` (interface), `Message` (universal role/content struct).
- **Implementations**: `pkg/ollama` and `pkg/cerebras`.

#### [`pkg/history`](./pkg/history)

**The Memory/Data Store.**

- **Purpose**: SQLite-based storage for message history and media metadata.
- **Main Methods**: `SaveMessage`, `SaveMedia`, `GetMessagesSince` (used to feed context to the LLM).

#### [`pkg/prompt`](./pkg/prompt)

**Prompt Management.**

- **Purpose**: Manages template-based prompts for different modes (Chat, Task, Behavior, Watcher).
- **Main Types**: `PromptManager`, `ModeData`, `BehaviorData`.

#### [`pkg/buttons`](./pkg/buttons)

**Interactive UI.**

- **Purpose**: Handles registry and resolution of interactive WhatsApp buttons.

#### [`pkg/agent` & `pkg/workflow`](./pkg/agent)

**Automation Framework.**

- **Purpose**: Primitives for building multi-step, stateful workflows that can be managed per conversation.

---

## Technical Flow

1. **Inbox**: `main.go` receives a WhatsApp event.
2. **Setup Check**: If the user is in a setup state, `pkg/batata` handles it.
3. **Watcher**: The message is checked against "Watcher rules" to decide if the bot should intervene.
4. **Intelligence**: `pkg/bot` (or `ProcessTask`) is called, fetching context from `pkg/history` and the prompt from `pkg/prompt`.
5. **Action**: The LLM returns JSON actions, which are executed via `pkg/bot/actions`.
