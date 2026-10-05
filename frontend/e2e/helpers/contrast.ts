export type RgbColor = { r: number; g: number; b: number; a: number }

const SRGB_CHANNEL_PATTERN = /^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)(?:\s*[,/]\s*([\d.%]+))?\s*\)$/
const COLOR_SRGB_PATTERN = /^color\(srgb\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)(?:\s*\/\s*([\d.%]+))?\s*\)$/

const parseAlpha = (rawAlpha: string | undefined): number => {
  if (rawAlpha === undefined) return 1
  if (rawAlpha.endsWith("%")) return Math.min(Math.max(parseFloat(rawAlpha) / 100, 0), 1)
  return Math.min(Math.max(parseFloat(rawAlpha), 0), 1)
}

export const parseColor = (color: string): RgbColor | null => {
  const srgbMatch = color.match(SRGB_CHANNEL_PATTERN)
  if (srgbMatch !== null) {
    return {
      r: parseFloat(srgbMatch[1]),
      g: parseFloat(srgbMatch[2]),
      b: parseFloat(srgbMatch[3]),
      a: parseAlpha(srgbMatch[4])
    }
  }

  const colorMatch = color.match(COLOR_SRGB_PATTERN)
  if (colorMatch !== null) {
    return {
      r: parseFloat(colorMatch[1]) * 255,
      g: parseFloat(colorMatch[2]) * 255,
      b: parseFloat(colorMatch[3]) * 255,
      a: parseAlpha(colorMatch[4])
    }
  }

  return null
}

export const compositeOver = (foreground: RgbColor, background: RgbColor): RgbColor => ({
  r: foreground.r * foreground.a + background.r * (1 - foreground.a),
  g: foreground.g * foreground.a + background.g * (1 - foreground.a),
  b: foreground.b * foreground.a + background.b * (1 - foreground.a),
  a: 1
})

export const relativeLuminance = (color: RgbColor): number => {
  const [red, green, blue] = [color.r, color.g, color.b].map((channel) => {
    const normalized = channel / 255
    return normalized <= 0.03928 ? normalized / 12.92 : Math.pow((normalized + 0.055) / 1.055, 2.4)
  })
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

export const contrastRatio = (foreground: RgbColor, background: RgbColor, pageBackground?: RgbColor): number => {
  const base = pageBackground ?? { r: 0, g: 0, b: 0, a: 1 }
  const opaqueBackground = background.a < 1 ? compositeOver(background, base) : background
  const opaqueForeground = foreground.a < 1 ? compositeOver(foreground, opaqueBackground) : foreground
  const lighter = Math.max(relativeLuminance(opaqueForeground), relativeLuminance(opaqueBackground))
  const darker = Math.min(relativeLuminance(opaqueForeground), relativeLuminance(opaqueBackground))
  return (lighter + 0.05) / (darker + 0.05)
}
