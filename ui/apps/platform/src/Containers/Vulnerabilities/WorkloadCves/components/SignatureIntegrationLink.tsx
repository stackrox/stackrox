import { Link } from 'react-router-dom';
import {
    getIntegrationsListPath,
    signatureIntegrationsSource as source,
    signatureIntegrationDescriptor as descriptor,
} from 'Containers/Integrations/utils/integrationsList';

import usePermissions from 'hooks/usePermissions';

import type { SignatureVerificationResult } from '../../types';

export type SignatureIntegrationLinkProps = {
    result: SignatureVerificationResult;
};

function SignatureIntegrationLink({ result }: SignatureIntegrationLinkProps) {
    const { hasReadAccess } = usePermissions();
    const displayName = result.verifierName || result.verifierId;
    const { type } = descriptor;
    const detailsUrl = `${getIntegrationsListPath(source, type)}/view/${result.verifierId}`;

    if (hasReadAccess('Integration')) {
        return <Link to={detailsUrl}>{displayName}</Link>;
    }

    return <>{displayName}</>;
}

export default SignatureIntegrationLink;
