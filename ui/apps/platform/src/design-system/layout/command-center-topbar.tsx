import { useState, useEffect } from 'react';
import { Search } from 'lucide-react';

import { cn } from 'design-system/lib/utils';

interface CommandCenterTopbarProps {
    title: string;
    onCommandPaletteOpen?: () => void;
}

export function CommandCenterTopbar({ title, onCommandPaletteOpen }: CommandCenterTopbarProps) {
    const [lastUpdated, setLastUpdated] = useState<string>('just now');

    useEffect(() => {
        const interval = setInterval(() => {
            setLastUpdated(
                new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
            );
        }, 60000);
        return () => clearInterval(interval);
    }, []);

    return (
        <header className="flex h-12 shrink-0 items-center gap-3 border-b border-border-subtle bg-bg-secondary px-5">
            <span className="text-sm font-600 text-text-primary">{title}</span>
            <div className="h-5 w-px bg-border" />

            <div className="ml-auto flex items-center gap-3">
                <button
                    type="button"
                    onClick={onCommandPaletteOpen}
                    className={cn(
                        'flex items-center gap-2 rounded-md border border-border bg-bg-tertiary px-3 py-1.5 text-xs text-text-muted transition-colors hover:border-text-muted'
                    )}
                >
                    <Search className="h-3 w-3" />
                    Search or jump to...
                    <kbd className="ml-1 rounded border border-border bg-bg-elevated px-1.5 py-0.5 font-mono text-2xs text-text-muted">
                        ⌘K
                    </kbd>
                </button>
                <span className="font-mono text-2xs text-text-muted">{lastUpdated}</span>
            </div>
        </header>
    );
}
