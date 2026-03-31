# E2E Scenario: Developer Assistant

## Steps

- [x] 1. Go server starts successfully, auto-indexes docs/*.md (5 files indexed with "ready" status)
- [x] 2. MCP devtools server connected (5 tools: dev_git_branch, dev_git_status, dev_git_log, dev_list_files, dev_read_file)
- [x] 3. dev_git_branch returns "master", dev_list_files returns 5 doc files
- [x] 4. /help command: RAG pipeline (rewrite/embed/search/filter/inject) all steps work
- [x] 5. /help returns answer with citations [N] and <!-- sources --> block from project documentation
- [x] 6. Frontend compiles (vue-tsc --noEmit), Go builds, devtools MCP binary builds
- [x] 7. Documents page shows auto-indexed docs/*.md files with "ready" status (verified via API)
