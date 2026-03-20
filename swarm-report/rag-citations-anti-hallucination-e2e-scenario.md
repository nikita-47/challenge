# E2E Scenario: RAG Citations, Sources & Anti-Hallucination

## Pre-conditions
- Dev servers running (Go + Vite)
- Document uploaded and indexed
- Ollama running with nomic-embed-text

## Steps

- [x] 1. Open http://localhost:5173, verify chat loads ✅
- [x] 2. Open settings popover, select an indexed document in Documents section ✅ (economic-research-chatgpt-usage-paper.pdf selected)
- [x] 3. Set RAG threshold to 0 (no filtering), ask a question about the document ✅ (asked: "What are the main findings about ChatGPT usage patterns in the economic research paper?")
- [x] 4. Verify: RAG pipeline steps appear (rewrite/embed/search/filter/inject) ✅ (all 5 steps visible: Rewrite → Embed → Search → Filter → Inject)
- [x] 5. Verify: assistant response contains inline citations [1], [2] etc rendered as small badges ✅ (citations found: [1], [2], [4] as badge elements with similarity scores)
- [x] 6. Verify: sources block appears below the response with ref numbers, filenames, chunk IDs, scores ✅ (sources (3) block with ref 1, 2, 4; economic-research-chatgpt-usage-paper.pdf; chunk IDs; scores: 63%, 62%, 62%)
- [x] 7. Verify: RAG context collapsible still works (chunks from N docs) ✅ (RAG context block shows 5 chunks from 1 doc with scores)
- [x] 8. Set high threshold (e.g. 0.95) and ask a question that will likely get filtered ✅ (threshold set to 0.95, asked: "What is the weather like in Tokyo today?")
- [x] 9. Verify: anti-hallucination amber banner appears ("No relevant documents found") ✅ (amber banner with "!" icon: "No relevant documents found -- response based on anti-hallucination rules")
- [x] 10. Verify: model response says it cannot answer based on available documents ✅ (response: "Cannot Answer Based on Available Documents" - explains no relevant content meets threshold, provides suggestions)
