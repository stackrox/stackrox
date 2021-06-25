import React, { ReactElement } from 'react';

import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import { ClusterInitBundle } from 'services/ClustersService';
import { Integration } from 'Containers/Integrations/integrationUtils';
import APITokensModal from './APITokens/APITokensModal';
import ClusterInitBundlesModal from './ClusterInitBundles/ClusterInitBundlesModal';

type ApiTokenType = {
    name: string;
    role: string;
};

type GenericIntegrationModalProps = {
    integrations: (Integration | ApiTokenType | ClusterInitBundle)[];
    source: string;
    type: string;
    label: string;
    onRequestClose: () => void;
    selectedIntegration: Integration | Record<string, unknown> | null;
};

const GenericIntegrationModal = ({
    integrations,
    source,
    type,
    label,
    onRequestClose,
    selectedIntegration,
}: GenericIntegrationModalProps): ReactElement => {
    if (source === 'authProviders' && type === 'apitoken') {
        const tokens = integrations as ApiTokenType[];
        return (
            <APITokensModal
                tokens={tokens}
                onRequestClose={onRequestClose}
                selectedIntegration={selectedIntegration}
            />
        );
    }

    if (source === 'authProviders' && type === 'clusterInitBundle') {
        const clusterInitBundles = integrations as ClusterInitBundle[];
        return (
            <ClusterInitBundlesModal
                clusterInitBundles={clusterInitBundles}
                onRequestClose={onRequestClose}
                selectedIntegration={selectedIntegration}
            />
        );
    }

    return (
        <IntegrationModal
            integrations={integrations}
            source={source}
            type={type}
            label={label}
            onRequestClose={onRequestClose}
            selectedIntegration={selectedIntegration}
        />
    );
};

export default GenericIntegrationModal;
