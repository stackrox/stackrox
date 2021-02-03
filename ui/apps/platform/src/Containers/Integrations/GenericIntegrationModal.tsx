import React, { ReactElement } from 'react';

import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import { ClusterInitBundle } from 'services/ClustersService';
import APITokensModal from './APITokens/APITokensModal';
import ClusterInitBundlesModal from './ClusterInitBundles/ClusterInitBundlesModal';

type ApiTokenType = {
    name: string;
    role: string;
};

type GenericIntegrationModalType = {
    apiTokens: ApiTokenType[];
    clusterInitBundles: ClusterInitBundle[];
    fetchEntitiesAndCloseModal: () => void;
    findIntegrations: (source, type) => ApiTokenType[] | ClusterInitBundle[];
    selectedTile: {
        type: string;
        source: string;
        label: string;
    };
};

const GenericIntegrationModal = ({
    apiTokens,
    clusterInitBundles,
    fetchEntitiesAndCloseModal,
    findIntegrations,
    selectedTile,
}: GenericIntegrationModalType): ReactElement => {
    const { source: selectedSource, type: selectedType, label: selectedLabel } = selectedTile;
    if (selectedSource === 'authProviders' && selectedType === 'apitoken') {
        return <APITokensModal tokens={apiTokens} onRequestClose={fetchEntitiesAndCloseModal} />;
    }

    if (selectedSource === 'authProviders' && selectedType === 'clusterInitBundle') {
        return (
            <ClusterInitBundlesModal
                clusterInitBundles={clusterInitBundles}
                onRequestClose={fetchEntitiesAndCloseModal}
            />
        );
    }

    const integrations = findIntegrations(selectedSource, selectedType);
    return (
        <IntegrationModal
            integrations={integrations}
            source={selectedSource}
            type={selectedType}
            label={selectedLabel}
            onRequestClose={fetchEntitiesAndCloseModal}
        />
    );
};

export default GenericIntegrationModal;
