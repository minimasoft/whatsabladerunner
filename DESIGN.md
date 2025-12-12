# Architecture

## Code

This is simple: event based with concurrency and trying to apply good golang practices (I know little of them so probably not following them).

There are only two key parts:

### 1. Integration with whatsmeow for whatsapp comms

This just follows events from whatsmeow and forwards what's required to the bot core.

### 2. Bot core

The bot core will apply pre-set prompt workflows to the events received and forward the responses to whatsmeow.

Each step of the state machine that represents a workflow will be a event to allow extensibility and easy testing.

Event trees should be documented.

## Governance

whatsabladerunner, or "Blady" from now on, is governed by the user using three key elements:

1. Tasks
2. Behaviors
3. Watchers

### 1. Tasks

The user asks Blady to talk to a contact or group with a specific goal.

Blady will create a task and confirm it with the user.

Once a task is confirmed Blady will gather information (read the conversation log, ask for missing information, etc.) and start chatting.

After every response Blady will evaluate and document task progress and continue chatting until the task is finished or blocked.

### 2. Behaviors

The user programs Blady with behaviors and then can enable those behaviors in conversations.

Once a behavior is enabled Blady will evaluate every message in the conversation with it until disabled.

### 3. Watchers

Blady is not responsible for its actions "he is just following orders".

Watchers will inspect every action Blady wants to execute (includes messages to send) and evaluate if it should be allowed to proceed or not.

If a watcher decides to block an action it will inform to the user the action blocked and the reason.

The user can decide to:
a. Accept the block and stop Blady from proceeding.
b. Ask Blady to try again.
c. Overrule the watcher and allow the action to proceed.

Exceptionally a task can be programmed to always do A or B but not C (well, actually you can but it will be highly discouraged).

## Config and execution

The config (tasks, behavoirs, watchers) will be stored in plain text in a structured directory.

While new tasks and behaviors can be created by the user talking to Blady, watchers can only be modified by the user directly editing the files.

whatsabladerunner will run locally from the terminal and by default will use local LLMs and sqlite storage.

The user can configure remote LLM providers.