import DOMPurify from "dompurify"

const MERMAID_LANGUAGE = "mermaid"
const ARROW_LANGUAGES = ["flow", "text", "plain", "txt"]
const DIAGRAM_LANGUAGES = [MERMAID_LANGUAGE, ...ARROW_LANGUAGES]
const FENCE_PATTERN = /^\s{0,3}(`{3,}|~{3,})\s*([A-Za-z0-9_-]*)\s*$/

export interface MarkdownSegment {
  type: "markdown" | "diagram"
  content: string
  language?: string
}

const normalizeLanguage = (language: string): string => language.trim().toLowerCase()

const isDiagramLanguage = (language: string): boolean =>
  DIAGRAM_LANGUAGES.includes(normalizeLanguage(language))

const escapeLabel = (label: string): string =>
  label
    .replace(/"/g, "#quot;")
    .replace(/\[/g, "#91;")
    .replace(/\]/g, "#93;")
    .replace(/[<>]/g, "")
    .trim()

const arrowChainToMermaid = (source: string): string => {
  const lines = source
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0 && !line.startsWith("#"))

  const edges: string[] = []
  const declared = new Set<string>()

  const nodeId = (label: string): string => {
    const id = `n${declared.size}`
    declared.add(label)
    return id
  }
  const nodeIds = new Map<string, string>()
  const idFor = (label: string): string => {
    const existing = nodeIds.get(label)
    if (existing !== undefined) return existing
    const id = nodeId(label)
    nodeIds.set(label, id)
    edges.push(`  ${id}["${escapeLabel(label)}"]`)
    return id
  }

  for (const line of lines) {
    const steps = line
      .split(/\s*(?:->|→|=>)\s*/)
      .map((step) => step.trim())
      .filter((step) => step.length > 0)
    for (let index = 1; index < steps.length; index += 1) {
      edges.push(`  ${idFor(steps[index - 1])} --> ${idFor(steps[index])}`)
    }
  }

  if (nodeIds.size === 0) return ""
  return `flowchart TD\n${edges.join("\n")}`
}

export const toMermaidSource = (source: string, language: string): string => {
  const normalized = normalizeLanguage(language)
  if (normalized === MERMAID_LANGUAGE) return source
  if (ARROW_LANGUAGES.includes(normalized)) return arrowChainToMermaid(source)
  return ""
}

export const parseMarkdownSegments = (source: string): MarkdownSegment[] => {
  const segments: MarkdownSegment[] = []
  const lines = source.split("\n")
  let markdownBuffer: string[] = []
  let diagramBuffer: string[] | null = null
  let diagramLanguage = ""
  let fenceMarker = ""

  const flushMarkdown = (): void => {
    if (markdownBuffer.length === 0) return
    const content = markdownBuffer.join("\n").trim()
    if (content.length > 0) segments.push({ type: "markdown", content })
    markdownBuffer = []
  }

  const flushDiagram = (): void => {
    if (diagramBuffer === null) return
    const content = diagramBuffer.join("\n").trim()
    if (content.length > 0) segments.push({ type: "diagram", content, language: diagramLanguage })
    diagramBuffer = null
  }

  for (const line of lines) {
    const fence = line.match(FENCE_PATTERN)

    if (diagramBuffer !== null) {
      if (fence && line.trim().startsWith(fenceMarker)) {
        flushDiagram()
        continue
      }
      diagramBuffer.push(line)
      continue
    }

    if (fence && isDiagramLanguage(fence[2] ?? "")) {
      flushMarkdown()
      fenceMarker = fence[1]
      diagramLanguage = normalizeLanguage(fence[2] ?? "")
      diagramBuffer = []
      continue
    }

    markdownBuffer.push(line)
  }

  flushDiagram()
  flushMarkdown()
  return segments
}

const sanitizeDiagramSvg = (svg: string): string =>
  DOMPurify.sanitize(svg, {
    USE_PROFILES: { svg: true, svgFilters: true },
    FORBID_TAGS: ["script", "foreignObject", "iframe", "object", "embed"],
    FORBID_ATTR: ["onload", "onerror", "onclick", "onmouseover", "onfocus"],
    ADD_ATTR: ["marker-end", "marker-start", "dominant-baseline"]
  })

type MermaidModule = typeof import("mermaid")

let mermaidModule: Promise<MermaidModule> | null = null
let renderSequence = 0

const loadMermaid = (): Promise<MermaidModule> => {
  if (mermaidModule === null) {
    mermaidModule = import("mermaid")
  }
  return mermaidModule
}

const DIAGRAM_KIND_LABELS: Record<string, string> = {
  sequenceDiagram: "Sequence",
  flowchart: "Flowchart",
  graph: "Flowchart",
  classDiagram: "Class diagram",
  erDiagram: "ER diagram",
  stateDiagram: "State diagram",
  "stateDiagram-v2": "State diagram",
  journey: "User journey",
  gantt: "Gantt",
  pie: "Pie chart",
  mindmap: "Mind map",
  timeline: "Timeline"
}

const readFrontmatterTitle = (source: string): string => {
  const block = source.match(/^---\s*\n([\s\S]*?)\n---/)
  if (block === null) return ""
  const line = block[1].split("\n").find((entry) => entry.trim().toLowerCase().startsWith("title:"))
  if (line === undefined) return ""
  return line.slice(line.indexOf(":") + 1).trim().replace(/^["']|["']$/g, "")
}

const readSequenceSubject = (source: string): string => {
  const participant = source.match(/^\s*participant\s+\S+\s+as\s+(.+)$/m)
  if (participant !== null) return participant[1].trim()
  const actor = source.match(/^\s*actor\s+\S+\s+as\s+(.+)$/m)
  if (actor !== null) return actor[1].trim()
  const arrow = source.match(/^\s*(\S+?)\s*(?:->>|-->>|->|-->)\s*(\S+?)\s*:/m)
  if (arrow !== null) return `${arrow[1]} to ${arrow[2]}`
  return ""
}

const readFlowchartSubject = (source: string): string => {
  const labelled = source.match(/\[["']?([^\]"']+)["']?\]/)
  if (labelled !== null) return labelled[1].trim()
  return ""
}

export const extractDiagramTitle = (source: string, language: string): string => {
  const explicit = readFrontmatterTitle(source)
  if (explicit.length > 0) return explicit

  const normalized = normalizeLanguage(language)
  if (normalized !== MERMAID_LANGUAGE) return "Flow"

  const kindMatch = source.match(/^\s*(sequenceDiagram|flowchart|graph|classDiagram|erDiagram|stateDiagram-v2|stateDiagram|journey|gantt|pie|mindmap|timeline)/m)
  const kind = kindMatch === null ? "" : kindMatch[1]
  const label = DIAGRAM_KIND_LABELS[kind] ?? "Diagram"

  const subject = kind === "sequenceDiagram" ? readSequenceSubject(source) : readFlowchartSubject(source)
  return subject.length > 0 ? `${label}: ${subject}` : label
}

const removeMermaidErrorNode = (renderId: string): void => {
  if (typeof document === "undefined") return
  for (const id of [renderId, `d${renderId}`]) {
    const element = document.getElementById(id)
    if (element !== null && element.parentElement !== null) {
      element.parentElement.removeChild(element)
    }
  }
}

export const renderDiagram = async (source: string, isDark: boolean): Promise<string> => {
  const module = await loadMermaid()
  const mermaid = module.default

  mermaid.initialize({
    startOnLoad: false,
    securityLevel: "strict",
    suppressErrorRendering: true,
    theme: isDark ? "dark" : "neutral",
    fontFamily: "ui-sans-serif, system-ui, -apple-system, sans-serif",
    flowchart: { htmlLabels: false },
    htmlLabels: false
  })

  renderSequence += 1
  const renderId = `lunar-diagram-${renderSequence}`
  try {
    const { svg } = await mermaid.render(renderId, source)
    return sanitizeDiagramSvg(svg)
  } catch (thrownError) {
    removeMermaidErrorNode(renderId)
    throw thrownError
  }
}
