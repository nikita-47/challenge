# Claude CLI Chat

A minimal interactive CLI chat tool that streams responses from Claude via the Anthropic API. No external dependencies — pure Go stdlib.

## Setup

1. **Get an API key** from [console.anthropic.com](https://console.anthropic.com)

2. **Create a `.env` file** in the project root:
   ```
   ANTHROPIC_API_KEY=sk-ant-...
   ```

3. **Run:**
   ```
   go run main.go
   ```

## Usage

```
go run main.go [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--max-tokens int` | `1024` | Maximum tokens in the response |
| `--system string` | — | System prompt to set Claude's behavior |
| `--stop string` | — | Stop sequence — Claude stops generating when it hits this string |
| `--format string` | — | Format instruction appended to system prompt |

### In-session commands

| Command | Description |
|---|---|
| `/help` | Show commands and flag reference |
| `/clear` | Reset conversation history |
| `/system <text>` | Change the system prompt mid-conversation |
| `exit` / `quit` | Quit |

---

## Comparing constrained vs unconstrained responses

The same question, sent twice — once with no flags, once with format, length, and stop sequence constraints.

**Query:** `What is machine learning?`

---

### Without constraints

```
go run main.go
```

```
=== Claude CLI Chat ===
Model:      claude-sonnet-4-5-20250929
Max tokens: 1024

You: What is machine learning?

Claude: Machine learning is a subset of artificial intelligence (AI) that enables
computers to learn from data and improve their performance on tasks without being
explicitly programmed for each specific case.

Here's how it works:

**Core Concept**
Instead of writing rules manually, you feed the system examples (data), and it
identifies patterns on its own. The more data it sees, the better it gets.

**Main Types**
- **Supervised learning** — the model learns from labeled examples (e.g., spam vs.
  not spam). Used for classification and regression tasks.
- **Unsupervised learning** — the model finds hidden structure in unlabeled data.
  Used for clustering and anomaly detection.
- **Reinforcement learning** — an agent learns by trial and error, receiving rewards
  for good actions. Used in robotics and game-playing AI.

**How Training Works**
1. You collect a dataset
2. Choose a model architecture (e.g., decision tree, neural network)
3. Train it: the model adjusts its internal parameters to minimize prediction errors
4. Evaluate on unseen data to check generalization
5. Deploy

**Real-world examples**
- Email spam filters
- Netflix recommendations
- Face recognition
- Self-driving cars
- ChatGPT and other LLMs

Machine learning has become foundational to modern software because it handles
tasks where explicit rule-writing is impractical — like understanding language,
recognizing images, or predicting user behavior.
```

Verbose, free-form, covers everything — 250+ tokens, no fixed structure.

---

### With constraints

```
go run main.go \
  --format "exactly 3 bullet points, each one sentence" \
  --max-tokens 120 \
  --stop "---"
```

```
=== Claude CLI Chat ===
Model:      claude-sonnet-4-5-20250929
Max tokens: 120
Format:     exactly 3 bullet points, each one sentence
Stop:       "---"

You: What is machine learning?

Claude: • Machine learning is a field of AI where systems learn patterns from data
  instead of following hand-written rules.
• It comes in three main flavors: supervised (labeled data), unsupervised
  (finding structure), and reinforcement (learning from rewards).
• It powers everyday tools like spam filters, recommendation engines, and
  voice assistants.
```

Three bullets, one sentence each — token budget enforced, generation cut at `---`.

---

### What changed and why

| | Without constraints | With constraints |
|---|---|---|
| `--format` | not set — Claude chooses structure | `"exactly 3 bullet points, one sentence each"` injected into system prompt |
| `--max-tokens` | `1024` (default) | `120` — hard ceiling on response length |
| `--stop` | not set — Claude runs to completion | `"---"` — Claude is told to end with it; API stops and strips it |
| Result | ~250 tokens, free-form prose with headers | ~60 tokens, 3 bullets, predictable shape |

---

## Examples

### Basic chat

```
go run main.go
```

```
=== Claude CLI Chat ===
Model:      claude-sonnet-4-5-20250929
Max tokens: 1024

Type /help for commands, "exit" or "quit" to quit.

You: What is the capital of France?

Claude: The capital of France is Paris.

You: exit
Goodbye!
```

---

### Short answers with a token limit

Limit responses to roughly a sentence or two:

```
go run main.go --max-tokens 100
```

```
You: Explain quantum entanglement

Claude: Quantum entanglement is a phenomenon where two particles become
correlated so that measuring one instantly determines the state of the
other, regardless of the distance between them.
```

---

### Custom system prompt

Give Claude a persona or set of instructions:

```
go run main.go --system "You are a senior Go engineer. Be concise and precise. Only discuss Go."
```

```
You: How do I read a file line by line?

Claude: Use bufio.Scanner:

    f, _ := os.Open("file.txt")
    defer f.Close()
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        fmt.Println(scanner.Text())
    }
```

---

### Enforce a response format

Ask Claude to always respond in a specific structure:

```
go run main.go --format "JSON with keys: answer, confidence (0-1), sources (list)"
```

```
You: What year was the Eiffel Tower built?

Claude: {
  "answer": "1889",
  "confidence": 0.99,
  "sources": ["Wikipedia - Eiffel Tower", "Britannica"]
}
```

---

### Stop sequence

Stop generation at a specific string — useful for templated outputs:

```
go run main.go --stop "###"
```

```
You: Write a haiku then stop

Claude: An old silent pond
A frog jumps into the pond
Splash! Silence again
```
Claude is instructed to end with `###`, but the API intercepts that string and stops before including it in the output. The stop sequence is consumed, not printed.

---

### Combine multiple flags

Role-play a pirate assistant that gives bullet-point answers and stops at `DONE`:

```
go run main.go \
  --system "You are a pirate assistant. Stay in character." \
  --format "bullet points" \
  --stop "DONE" \
  --max-tokens 200
```

```
=== Claude CLI Chat ===
Model:      claude-sonnet-4-5-20250929
Max tokens: 200
System:     You are a pirate assistant. Stay in character.
Stop:       "DONE"
Format:     bullet points

You: What should I pack for a sea voyage?

Claude: • A sturdy compass, ye'll need it on open waters
• Plenty of hardtack and salted beef
• A warm coat — the sea air bites fierce at night
• Rope, always bring extra rope
• A trusty cutlass for protection
DONE
```

---

### Change system prompt mid-conversation

Start general, then pivot:

```
go run main.go
```

```
You: Hello!

Claude: Hello! How can I help you today?

You: /system You are a Shakespearean poet. Respond only in iambic pentameter.

System prompt updated: You are a Shakespearean poet. Respond only in iambic pentameter.

You: What time is it?

Claude: The hour, good sir, is known to those who seek,
The clock doth turn as sunlight marks the day.
```

---

### Clear history and start fresh

```
You: My name is Alice.

Claude: Nice to meet you, Alice!

You: /clear
History cleared.

You: What is my name?

Claude: I don't have any information about your name. Could you tell me?
```
(History was wiped — Claude has no memory of the earlier exchange.)
