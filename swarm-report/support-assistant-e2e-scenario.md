# E2E Scenario: Support Assistant Widget

## Steps

- [x] 1. Open the app at localhost:5173 (or 5174) — verify the floating "?" button is visible at bottom-left ✅
- [x] 2. Click the "?" button — verify the support chat panel opens (500px tall, 384px wide, with header "Support" and input area) ✅
- [x] 3. Verify empty state shows "How can I help you?" text ✅
- [x] 4. Type "Hello, how do I start a new chat?" and press Enter — verify message appears as user bubble (right side, primary color) ✅
- [x] 5. Wait for AI response to stream — verify assistant bubble appears (left side, muted color) with relevant FAQ content ✅
- [x] 6. Verify the minimize button (−) closes the panel back to "?" circle ✅
- [x] 7. Click "?" again, verify previous messages are still visible (state preserved) ✅
- [x] 8. Navigate to another view (e.g., docs) — verify "?" button is still visible on that view ✅
- [x] 9. Click "?" on docs view — verify support chat works there too ✅
