# FAQ: General Questions

## What is this application?

This is an AI Chat application built as a 35-day challenge. It provides a conversational interface powered by Claude (Anthropic's AI), with features like multi-session management, context strategies, memory layers, RAG document search, MCP tool integration, code review automation, and more.

## How do I start a new chat session?

Click the "New Chat" button in the left sidebar (Sessions panel). Enter a name for your session and click Create. The new session will appear in the session list.

## How do I switch between chat sessions?

Click on any session name in the left sidebar to load that session. Your conversation history is preserved per session.

## How do I delete a chat session?

Hover over a session in the left sidebar and click the delete (trash) icon. This permanently removes the session and its history.

## What AI models are available?

The application supports three providers:
- **Claude** (Anthropic) — Claude Sonnet and Claude Haiku models via the Anthropic API
- **Local LLM** — Any OpenAI-compatible local model (e.g., via LM Studio or Ollama)
- **Railway** — Remote Ollama instance hosted on Railway

You can switch providers in the settings panel.

## How do I change the AI provider?

Open the settings panel by clicking the gear icon. Select your preferred provider (Claude, Local, or Railway). For local and Railway providers, you may need to configure the URL, model name, and API key.

## What is the temperature setting?

Temperature controls the randomness of AI responses. Lower values (0.0-0.3) produce more focused, deterministic answers. Higher values (0.7-1.0) produce more creative, varied responses. Default is 1.0.

## How do I use the /help command?

Type `/help` followed by your question in the chat input. For example: `/help How do RAG documents work?`. The /help command automatically searches the project documentation using RAG and provides contextual answers.

## What are context strategies?

Context strategies control how conversation history is managed:
- **Window** — keeps the last N messages
- **Summary** — compresses older messages into summaries
- **Facts** — extracts key facts from the conversation
- **Branch** — creates conversation branches for exploring different topics
