import React from 'react';
import usePermissions from 'hooks/usePermissions';
import LoadingSection from 'Components/PatternFly/LoadingSection';

function PluginContent({ children }: { children: React.ReactNode }) {
    const { isLoadingPermissions } = usePermissions();

    if (isLoadingPermissions) {
        return <LoadingSection />;
    }

    return <>{children}</>;
}

export default PluginContent;
