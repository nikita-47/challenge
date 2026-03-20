import type { RAGSource } from '@/lib/types'

/**
 * Parses the hidden sources block from assistant message content.
 * The backend appends a comment block in the form:
 *   <!-- sources
 *   [{"ref":1,"source":"file.pdf","chunk":"chunk_a3","score":0.82}]
 *   -->
 */
export function parseRAGSources(content: string): { cleanContent: string; sources: RAGSource[] } {
  const regex = /<!-- sources\n(\[[\s\S]*?\])\n-->/

  const match = regex.exec(content)
  if (!match) {
    return { cleanContent: content, sources: [] }
  }

  try {
    const sources = JSON.parse(match[1]!) as RAGSource[]
    const cleanContent = content.replace(regex, '').trimEnd()
    return { cleanContent, sources }
  } catch {
    console.error('[ragCitations] Failed to parse sources block:', match[1])
    return { cleanContent: content, sources: [] }
  }
}

/**
 * Replaces inline citation markers like [1], [2] in rendered HTML
 * with interactive badge spans. Skips citations inside <a>, <code>,
 * and <pre> tags to avoid corrupting links and code blocks.
 *
 * Only purely numeric references are matched — e.g. [1] but not [text](url).
 */
export function renderCitations(html: string, sources: RAGSource[]): string {
  if (!sources.length) {
    return html
  }

  const sourceMap = new Map<number, RAGSource>()
  for (const s of sources) {
    sourceMap.set(s.ref, s)
  }

  // Split HTML into segments: protected (inside tags like <a>, <code>, <pre>)
  // and unprotected (plain text nodes between tags).
  // Strategy: tokenize by tags, only transform text between tags that are
  // not inside a protected context.
  const result: string[] = []
  // Matches either a complete tag or a text segment between tags
  const tokenizer = /(<(?:a|code|pre)\b[^>]*>[\s\S]*?<\/(?:a|code|pre)>)|(<[^>]+>)|([^<]+)/gi

  let match: RegExpExecArray | null
  while ((match = tokenizer.exec(html)) !== null) {
    const protectedBlock = match[1]
    const tag = match[2]
    const text = match[3]

    if (protectedBlock !== undefined) {
      // Protected block — output as-is
      result.push(protectedBlock)
    } else if (tag !== undefined) {
      // Regular tag — output as-is
      result.push(tag)
    } else if (text !== undefined) {
      // Text segment — replace [N] citations
      result.push(replaceCitationsInText(text, sourceMap))
    }
  }

  return result.join('')
}

function replaceCitationsInText(text: string, sourceMap: Map<number, RAGSource>): string {
  // Match [N] where N is a pure integer, not followed by ( (which would be [text](url))
  return text.replace(/\[(\d+)\](?!\()/g, (fullMatch, numStr) => {
    const ref = parseInt(numStr, 10)
    const source = sourceMap.get(ref)

    if (!source) {
      return fullMatch
    }

    const scorePercent = Math.round(source.score * 100)
    const title = `${source.source} (${scorePercent}%)`

    return `<span class="rag-cite" data-ref="${ref}" title="${title}">${ref}</span>`
  })
}
