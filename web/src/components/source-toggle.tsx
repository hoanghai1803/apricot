import type { BlogSource } from '@/lib/types'
import { Switch } from '@/components/ui/switch'

interface SourceToggleProps {
  source: BlogSource
  onToggle: (id: number, active: boolean) => void
}

export function SourceToggle({ source, onToggle }: SourceToggleProps) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border p-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <span className="font-medium">{source.company}</span>
          <span className="text-sm text-muted-foreground">{source.name}</span>
        </div>
        <p className="mt-1 truncate text-xs text-muted-foreground">
          {source.feed_url}
        </p>
      </div>
      <Switch
        checked={source.is_active}
        onCheckedChange={(checked: boolean) => onToggle(source.id, checked)}
        aria-label={`Toggle ${source.company} - ${source.name}`}
      />
    </div>
  )
}
