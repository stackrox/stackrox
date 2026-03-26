import type { ReactElement } from 'react';

import type { Deployment } from 'types/deployment.proto';
import SecurityContextCard from 'Components/SecurityContextCard';

export type SecurityContextProps = {
    deployment: Deployment | null;
};

function SecurityContext({ deployment }: SecurityContextProps): ReactElement {
    const emptyMessage =
        deployment === null
            ? "Security context is unavailable because the alert's deployment no longer exists."
            : 'None';

    return <SecurityContextCard containers={deployment?.containers} emptyMessage={emptyMessage} />;
}

export default SecurityContext;
