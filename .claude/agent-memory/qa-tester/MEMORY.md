# QA Tester Memory Index

## Testing References

### MCP Servers
- [Pipeline MCP Testing](pipeline-mcp.md) - Architecture, test patterns, known issues (model deprecated bug)

## Test Notes

### Code Review Feature (Day 32) - PASS ✅
Tested full E2E flow with real GitHub PR #1. Key findings:
- ReviewView component pre-built, needed integration into App.vue
- Missing types (PullRequest, ReviewStep*) - easily added to types.ts
- Missing API function (fetchOpenPRs) - easily added to api.ts
- All 4 pipeline steps complete: diff → rag → analyze → comment
- Review quality excellent, markdown rendered correctly
- No console errors, all API calls 200 OK
- Feature production-ready
