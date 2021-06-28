import React, { useState, ReactElement, useEffect } from 'react';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';

import { actions } from 'reducers/clusterInitBundles';
import { selectors } from 'reducers';
import { ClusterInitBundle } from 'services/ClustersService';
import { Integration } from 'Containers/Integrations/utils/integrationUtils';

import Modal from 'Components/Modal';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import ClusterInitBundleForm from './ClusterInitBundleForm';
import ClusterInitBundleDetails from './ClusterInitBundleDetails';

export type ClusterInitBundlesModalProps = {
    authProviders: { name: string; id: string }[];
    clusterInitBundles: ClusterInitBundle[];
    clusterInitBundleGenerationWizardOpen: boolean;
    onRequestClose: () => void;
    startClusterInitBundleGenerationWizard: () => void;
    closeClusterInitBundleGenerationWizard: () => void;
    generateClusterInitBundle: () => void;
    currentGeneratedClusterInitBundle?: ClusterInitBundle | null;
    currentGeneratedHelmValuesBundle?: ClusterInitBundle | null;
    currentGeneratedKubectlBundle?: ClusterInitBundle | null;
    selectedIntegration: Integration | Record<string, unknown> | null;
};

function ClusterInitBundlesModal({
    authProviders = [],
    clusterInitBundles,
    onRequestClose,
    closeClusterInitBundleGenerationWizard,
    generateClusterInitBundle,
    currentGeneratedClusterInitBundle = null,
    currentGeneratedHelmValuesBundle = null,
    currentGeneratedKubectlBundle = null,
    selectedIntegration = null,
}: ClusterInitBundlesModalProps): ReactElement {
    const [selectedBundleId, setSelectedBundleId] = useState<string | null>(null);

    useEffect(() => {
        let id: string | null = null;
        if (selectedIntegration) {
            id = selectedIntegration?.id as string;
        }
        setSelectedBundleId(id);
    }, [selectedIntegration]);

    function onSubmit() {
        generateClusterInitBundle();
    }

    function closeModal() {
        closeClusterInitBundleGenerationWizard();
        onRequestClose();
    }

    function renderHeader() {
        return (
            <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
                <span className="flex flex-1">Configure Cluster Init Bundles</span>
                <Icon.X className="h-4 w-4 cursor-pointer" onClick={closeModal} />
            </header>
        );
    }

    function renderForm() {
        if (selectedBundleId) {
            return null;
        }

        const buttons = (
            <PanelButton
                icon={<Icon.Save className="h-4 w-4" />}
                className="btn btn-success mr-2"
                onClick={onSubmit}
                tooltip="Generate"
            >
                Generate
            </PanelButton>
        );

        return (
            <PanelNew testid="panel">
                <PanelHead>
                    <PanelTitle
                        isUpperCase
                        testid="panel-header"
                        text="Generate Cluster Init Bundle"
                    />
                    <PanelHeadEnd>{buttons}</PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <ClusterInitBundleForm />
                </PanelBody>
            </PanelNew>
        );
    }

    function renderDetails() {
        if (currentGeneratedClusterInitBundle) {
            return (
                <PanelNew testid="panel">
                    <PanelBody>
                        <ClusterInitBundleDetails
                            authProviders={authProviders}
                            clusterInitBundle={currentGeneratedClusterInitBundle}
                            helmValuesBundle={currentGeneratedHelmValuesBundle}
                            kubectlBundle={currentGeneratedKubectlBundle}
                        />
                    </PanelBody>
                </PanelNew>
            );
        }
        if (selectedBundleId) {
            const selectedBundleMetadata = clusterInitBundles.find(
                ({ id }) => id === selectedBundleId
            );
            if (selectedBundleMetadata) {
                return (
                    <PanelNew testid="panel">
                        <PanelBody>
                            <ClusterInitBundleDetails
                                authProviders={authProviders}
                                clusterInitBundle={selectedBundleMetadata}
                                helmValuesBundle={currentGeneratedHelmValuesBundle}
                                kubectlBundle={currentGeneratedKubectlBundle}
                            />
                        </PanelBody>
                    </PanelNew>
                );
            }
        }
        return null;
    }

    return (
        <Modal isOpen onRequestClose={onRequestClose} className="">
            {renderHeader()}
            <div className="flex flex-1 relative w-full bg-base-100">
                {renderForm()}
                {renderDetails()}
            </div>
        </Modal>
    );
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    clusterInitBundleGenerationWizardOpen: selectors.clusterInitBundleGenerationWizardOpen,
    currentGeneratedClusterInitBundle: selectors.getCurrentGeneratedClusterInitBundle,
    currentGeneratedHelmValuesBundle: selectors.getCurrentGeneratedHelmValuesBundle,
    currentGeneratedKubectlBundle: selectors.getCurrentGeneratedKubectlBundle,
});

const mapDispatchToProps = {
    startClusterInitBundleGenerationWizard: actions.startClusterInitBundleGenerationWizard,
    closeClusterInitBundleGenerationWizard: actions.closeClusterInitBundleGenerationWizard,
    generateClusterInitBundle: actions.generateClusterInitBundle.request as () => void,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterInitBundlesModal);
