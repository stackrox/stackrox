import type { ReactElement } from 'react';

import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { HasReadAccess } from 'hooks/usePermissions';

import { CommandCenterSidebar } from 'design-system/layout/command-center-sidebar';
import { CommandCenterTopbar } from 'design-system/layout/command-center-topbar';
import { CommandPalette, useCommandPalette } from 'design-system/layout/command-palette';

import Body from './Body';

interface CommandCenterShellProps {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
}

export function CommandCenterShell({
    hasReadAccess,
    isFeatureFlagEnabled,
}: CommandCenterShellProps): ReactElement {
    const { open, setOpen } = useCommandPalette();

    return (
        <div className="flex h-screen bg-bg-primary text-text-primary font-sans text-sm">
            <CommandCenterSidebar />
            <div className="flex flex-1 flex-col overflow-hidden">
                <CommandCenterTopbar
                    title="StackRox"
                    onCommandPaletteOpen={() => setOpen(true)}
                />
                <main className="flex-1 overflow-y-auto">
                    <ErrorBoundary>
                        <Body
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                    </ErrorBoundary>
                </main>
            </div>
            <CommandPalette open={open} onOpenChange={setOpen} />
        </div>
    );
}
