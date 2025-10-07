import React, { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    List,
    ListItem,
    Modal,
} from '@patternfly/react-core';

import useAnalytics, { REVOKE_INIT_BUNDLE } from 'hooks/useAnalytics';
import { revokeClusterInitBundles } from 'services/ClustersService';
import type { ClusterInitBundle, ImpactedCluster } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type RevokeBundleModalProps = {
    initBundle: ClusterInitBundle;
    onCloseModal: (wasRevoked: boolean) => void;
};

function RevokeBundleModal({ initBundle, onCloseModal }: RevokeBundleModalProps): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const [errorMessage, setErrorMessage] = useState('');
    const [impactedClusters, setImpactedClusters] = useState<ImpactedCluster[]>(
        initBundle.impactedClusters
    );
    const [hasMoreClusters, setHasMoreClusters] = useState(false);
    const [isRevokingBundle, setIsRevokingBundle] = useState(false);

    function onRevokeBundle() {
        setErrorMessage('');
        setIsRevokingBundle(true);
        revokeClusterInitBundles(
            [initBundle.id],
            impactedClusters.map(({ id }) => id)
        )
            .then(({ initBundleRevocationErrors }) => {
                if (initBundleRevocationErrors.length === 0) {
                    setHasMoreClusters(false);
                    onCloseModal(true);
                } else {
                    // The bundle has more impacted clusters than the list already rendered.
                    // Therefore, user needs to confirm revoke again.
                    setHasMoreClusters(true);
                    setImpactedClusters(initBundleRevocationErrors[0].impactedClusters);
                }
                analyticsTrack(REVOKE_INIT_BUNDLE);
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsRevokingBundle(false);
            });
    }

    function onCancel() {
        setErrorMessage('');
        setHasMoreClusters(false);
        onCloseModal(false);
    }

    // showClose={false} to prevent clicking close while isRevokingBundle.
    return (
        <Modal
            title="Revoke cluster init bundle"
            variant="small"
            isOpen
            showClose={false}
            actions={[
                <Button
                    key="Revoke bundle"
                    variant={impactedClusters.length === 0 ? 'primary' : 'danger'}
                    onClick={onRevokeBundle}
                    isDisabled={isRevokingBundle}
                >
                    Revoke bundle
                </Button>,
                <Button
                    key="Cancel"
                    variant="secondary"
                    onClick={onCancel}
                    isDisabled={isRevokingBundle}
                >
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <DescriptionList isHorizontal>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Bundle name</DescriptionListTerm>
                        <DescriptionListDescription>{initBundle.name}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
                {impactedClusters.length === 0 ? (
                    <Alert
                        title="No secured clusters depend on this bundle"
                        variant="info"
                        isInline
                        component="p"
                    />
                ) : (
                    <>
                        {hasMoreClusters ? (
                            <Alert
                                title="Additional secured clusters now depend on this initBundle"
                                variant="warning"
                                isInline
                                component="p"
                            >
                                You must confirm <strong>again</strong> to delete this bundle.
                            </Alert>
                        ) : (
                            <Alert
                                title="Secured clusters depend on this bundle"
                                variant="danger"
                                isInline
                                component="p"
                            >
                                <p>
                                    In clusters that depend on this bundle, secured cluster services
                                    like Sensor will lose connectivity to Central.
                                </p>
                                <p className="pf-v5-u-mt-md">
                                    We recommend that you <strong>replace</strong> this bundle in
                                    the following secured clusters <strong>before</strong> you
                                    revoke it.
                                </p>
                            </Alert>
                        )}
                        <List component="ol">
                            {impactedClusters.map(({ name }) => (
                                <ListItem key={name}>{name}</ListItem>
                            ))}
                        </List>
                    </>
                )}
                {errorMessage && (
                    <Alert title="Revoke bundle failed" variant="danger" isInline component="p">
                        {errorMessage}
                    </Alert>
                )}
            </Flex>
        </Modal>
    );
}

export default RevokeBundleModal;
