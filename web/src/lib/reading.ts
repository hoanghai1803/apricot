/** Format reading time as "5 min", "1 hr 15 min", etc. */
export function formatReadingTime(minutes?: number): string {
  if (!minutes || minutes <= 0) return ''
  if (minutes < 60) return `${minutes} min`
  const hrs = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (mins === 0) return `${hrs} hr`
  return `${hrs} hr ${mins} min`
}

/** Split plain text into paragraphs by double newlines. */
export function formatPlainText(text: string): string[] {
  return text
    .split(/\n{2,}/)
    .map((p) => p.trim())
    .filter(Boolean)
}
