import type { ReactNode } from 'react';

import { CommandCenterSidebar } from './command-center-sidebar';
import { CommandCenterTopbar } from './command-center-topbar';
import { CommandPalette, useCommandPalette } from './command-palette';

interface CommandCenterLayoutProps {
    title: string;
    children: ReactNode;
}

export function CommandCenterLayout({ title, children }: CommandCenterLayoutProps) {
    const { open, setOpen } = useCommandPalette();

    return (
        <div className="flex h-screen bg-bg-primary text-text-primary font-sans text-sm">
            <CommandCenterSidebar />
            <div className="flex flex-1 flex-col overflow-hidden">
                <CommandCenterTopbar title={title} onCommandPaletteOpen={() => setOpen(true)} />
                <main className="flex-1 overflow-y-auto">{children}</main>
            </div>
            <CommandPalette open={open} onOpenChange={setOpen} />
        </div>
    );
}
