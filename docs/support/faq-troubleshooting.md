# FAQ: Troubleshooting

## The chat is not responding / shows an error

Common causes:
1. **API key not set** — ensure `ANTHROPIC_API_KEY` is set in your `.env` file
2. **Backend not running** — run `./dev.sh start` to start both Go and Vite servers
3. **Provider mismatch** — if using Local/Railway provider, ensure the LLM server is running

Check the browser console (F12 → Console) for specific error messages.

## RAG is not finding relevant results

Possible solutions:
1. **Check document status** — go to Documents view and verify the document status is "ready"
2. **Lower the threshold** — try reducing the RAG threshold to 0.15-0.20
3. **Increase top-K** — allow more chunks to be retrieved (try 8-10)
4. **Enable query rewrite** — turn on the "Rewrite query" option in RAG settings
5. **Ollama not running** — RAG requires Ollama with `nomic-embed-text` model for embeddings. Start Ollama and ensure the model is available.

## MCP server won't connect

1. Ensure the MCP server binary is built: `./dev.sh start` builds all MCP servers
2. Check that `.mcp_servers.json` exists and has correct paths
3. Try disconnecting and reconnecting via the MCP tab
4. Check the Go server logs for MCP-related errors

## Local LLM is not working

1. Verify LM Studio or Ollama is running and the API endpoint is accessible
2. Check the URL in settings (default: `http://localhost:1234/v1` for LM Studio)
3. Ensure a model is loaded in your local LLM server
4. Try the "Railway" provider as an alternative remote LLM

## Document upload fails

1. Check file format — only .md, .pdf, and .txt are supported
2. Ensure the file is not too large (recommended: under 10MB)
3. Check that the `.documents/` directory exists and is writable
4. For PDFs, ensure the file is not password-protected

## The UI looks broken or components are missing

1. Clear browser cache and hard refresh (Ctrl+Shift+R)
2. Ensure Vite dev server is running: `./dev.sh start-vite`
3. Check for JavaScript errors in the browser console

## How do I report a bug?

Create a support ticket describing:
1. What you were trying to do
2. What happened instead
3. Any error messages you see
4. Steps to reproduce the issue

The support assistant can help create a ticket for you.
