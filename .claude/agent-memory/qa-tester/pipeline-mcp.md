---
name: Pipeline MCP Testing
description: QA notes on Pipeline MCP server - architecture, test patterns, known issues
type: reference
---

# Pipeline MCP Server - QA Reference

## Overview
Pipeline MCP is a 3-stage data processing server:
1. **search**: Query HackerNews Algolia API
2. **summarize**: Summarize results with Claude API
3. **save**: Write output to markdown file

## Architecture
- **Location**: `/Users/nbonachev/dev/challenge/mcp-servers/pipeline/`
- **Files**: main.go (6 tools), runner.go (execution), store.go (persistence)
- **Config**: `.mcp_servers.json` with environment vars
- **Data**: Stores pipeline runs in `pipeline_data.json` alongside binary

## Tools (6 total)
- `pipe_run`: Start async pipeline (returns run ID)
- `pipe_status`: Get run status with step details
- `pipe_list`: List all runs
- `pipe_search`: Direct HN search (no async)
- `pipe_summarize`: Direct summarize (no async)
- `pipe_save`: Direct save to file (no async)

## Testing Patterns

### Happy Path Test
```bash
# Start pipeline
curl -X POST /api/mcp/tools/call \
  -d '{
    "server": "pipeline",
    "tool": "pipe_run",
    "arguments": {"query": "test query", "count": 3}
  }'

# Poll status every 2 seconds until done/error
curl -X POST /api/mcp/tools/call \
  -d '{"server": "pipeline", "tool": "pipe_status", "arguments": {"id": "RUN_ID"}}'
```

### Expected Step Progression
```
pending → running → done  (search)
pending → running → done  (summarize) ← BLOCKED BY MODEL BUG
pending → running → done  (save)
```

## Known Issues

### 🔴 CRITICAL: Outdated Model (BLOCKING)
- **Location**: `runner.go` line 226
- **Current**: `"model": "claude-3-5-haiku-20241022"`
- **Problem**: Model no longer exists in Anthropic API (404 error)
- **Fix**: Update to `claude-3-5-haiku-20240307` or current model
- **Impact**: Summarize step always fails → entire pipeline fails

This is the ONLY issue blocking full E2E functionality.

## Frontend Integration
- **Component**: `/frontend/src/components/PipelinePanel.vue`
- **Store**: `/frontend/src/stores/pipeline.ts`
- **Polling**: Default 2000ms interval
- **Tab**: "pipeline" in SessionPanel.vue sidebar

## Test Results Summary (Updated 2026-03-13)
- **Search Step**: ✅ 100% pass rate (HN Algolia API works)
- **Summarize Step**: ✅ 100% pass rate (model bug FIXED - now working)
- **Save Step**: ✅ 100% pass rate (creates markdown files correctly)
- **Infrastructure**: ✅ 100% pass rate (MCP, API, store, UI all work)
- **UI Integration**: ✅ 100% pass rate (SessionPanel tab, PipelinePanel component)
- **Polling/State**: ✅ Works correctly (2000ms interval, proper stop on completion)

## Quick Verdict
Pipeline feature is PRODUCTION READY. All steps execute successfully, UI is fully integrated, no blocking issues.

## March 13 Test Results
Full E2E test performed: start pipeline → poll status → verify all steps complete → check output file
- 3 successful runs with "rust programming" query
- All 6 tools responding correctly
- Output files generated with proper markdown format
- Status polling working reliably
- UI components properly integrated into SessionPanel sidebar
