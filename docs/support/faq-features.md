# FAQ: Features Guide

## What is RAG (Retrieval-Augmented Generation)?

RAG enhances AI responses by searching through uploaded documents for relevant information. When enabled, the system finds document chunks related to your question and provides them as context to the AI, resulting in more accurate, grounded answers with citations.

## How do I use RAG?

1. Upload documents in the Documents view (click "documents" in the left sidebar)
2. Wait for indexing to complete (status changes to "ready")
3. In chat settings (gear icon), enable RAG by selecting documents and adjusting threshold/top-K
4. Ask questions — the AI will search your documents and cite sources

## What is the RAG threshold?

The threshold (0.0-1.0) controls how relevant a document chunk must be to be included. Higher values mean stricter filtering (only very relevant chunks). Lower values include more context but may add noise. Default is 0.25.

## What document formats are supported?

The application supports:
- **Markdown (.md)** — best supported, preserves structure
- **PDF (.pdf)** — text extraction with formatting
- **Plain text (.txt)** — basic text documents

## What is MCP (Model Context Protocol)?

MCP allows the AI to use external tools. The application includes several MCP servers:
- **HackerNews** — search and browse Hacker News stories
- **Scheduler** — create reminders, URL monitors, and scheduled tasks
- **Pipeline** — search, summarize, and save HN content
- **DevTools** — access project files, git status, and logs

## How do I enable MCP tools?

1. MCP servers are configured in `.mcp_servers.json`
2. Connect to a server via the MCP tab in settings
3. Enable specific tools in the chat settings (gear icon → MCP Servers section)
4. The AI can then use these tools during conversation

## What is the Pipeline feature?

Pipeline automates a search → summarize → save workflow for Hacker News content. Access it via "pipeline" in the left sidebar. Enter a search query, and the system fetches relevant articles, summarizes them with AI, and saves the output.

## What is Code Review?

The Code Review feature analyzes GitHub pull requests using AI. It fetches the PR diff, enriches it with project documentation context (via RAG), and generates a detailed code review that gets posted as a PR comment.

## What are Memory Layers?

The application has a multi-layer memory system:
- **Short-term** — current chat messages
- **Working memory** — project-specific notes (auto-updated)
- **Long-term** — user profile preferences (auto-updated)
- **Operator** — immutable system identity

Memory layers help the AI maintain context across sessions.

## What is Task Mode?

Task Mode enables a structured agent workflow for complex tasks. The AI proposes phases (planning → executing → validating), executes them step-by-step, and validates the results. Enable it via the checkbox in chat settings.

## What are Invariants?

Invariants are constraints that the AI agent must follow during Task Mode. They are checked before each action. Example: "Do not modify the database schema" or "All changes must be backward compatible."
