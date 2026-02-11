import { Link } from 'react-router-dom';
import { integrationsPath } from 'routePaths';

import usePermissions from 'hooks/usePermissions';

import type { SignatureVerificationResult } from '../../types';

export type SignatureIntegrationLinkProps = {
    result: SignatureVerificationResult;
};

function SignatureIntegrationLink({ result }: SignatureIntegrationLinkProps) {
    const { hasReadAccess } = usePermissions();
    const displayName = result.verifierName || result.verifierId;
    const detailsUrl = `${integrationsPath}/signatureIntegrations/signature/view/${result.verifierId}`;

    if (hasReadAccess('Integration')) {
        return <Link to={detailsUrl}>{displayName}</Link>;
    }

    return <>{displayName}</>;
}

export default SignatureIntegrationLink;
