import React, { ReactElement, useEffect, useRef, useState } from 'react';
import { Alert, Badge, Button, List, ListItem, Modal, ModalVariant } from '@patternfly/react-core';

import {
    ClusterInitBundle,
    ImpactedCluster,
    revokeClusterInitBundles,
} from 'services/ClustersService';

export type DeleteClusterInitBundleConfirmationModalProps = {
    bundle?: ClusterInitBundle;
    handleCancel: () => void;
    handleDelete: (bundleId: string) => void;
};

/*
 * The confirmation modal does not support multiple bundles by design,
 * so the relationship between the bundle and its impacted clusters is clear,
 * in case people need to replace the bundle in its impacted clusters.
 */
function DeleteClusterInitBundleConfirmationModal({
    bundle,
    handleCancel,
    handleDelete,
}: DeleteClusterInitBundleConfirmationModalProps): ReactElement {
    const [alert, setAlert] = useState<ReactElement | null>(null);
    const [impactedClusters, setImpactedClusters] = useState<ImpactedCluster[]>([]);
    const [isRaceCondition, setIsRaceCondition] = useState(false);
    const [isRevokingBundle, setIsRevokingBundle] = useState(false);
    const refCancelButton = useRef<null | HTMLButtonElement>(null);

    useEffect(() => {
        setImpactedClusters(bundle ? bundle.impactedClusters : []);

        /*
         * Cancel button has initial focus so delete requires an intentional action,
         * like click button or press tab key, not just press return key.
         */
        if (typeof refCancelButton?.current?.focus === 'function') {
            refCancelButton.current.focus();
        }
    }, [bundle]);

    const isOpen = Boolean(bundle);
    const bundleId = bundle ? bundle.id : '';
    const bundleName = bundle ? bundle.name : '';

    function onDelete() {
        setIsRevokingBundle(true);
        setAlert(null);
        revokeClusterInitBundles(
            [bundleId],
            impactedClusters.map(({ id }) => id)
        )
            .then(({ initBundleRevocationErrors }) => {
                if (initBundleRevocationErrors.length === 0) {
                    setIsRaceCondition(false);
                    handleDelete(bundleId); // in case the list ever manages its own state in the future
                } else {
                    setIsRaceCondition(true);
                    setImpactedClusters(initBundleRevocationErrors[0].impactedClusters);
                }
            })
            .catch((error) => {
                setAlert(
                    <Alert title="Delete bundle failed" variant="danger" isInline>
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsRevokingBundle(false);
            });
    }

    function onCancel() {
        setAlert(null);
        setIsRaceCondition(false);
        handleCancel();
    }

    // showClose={false} to prevent clicking close while isRevokingBundle.
    return (
        <Modal
            variant={ModalVariant.small}
            title="Delete bundle"
            isOpen={isOpen}
            showClose={false}
            actions={[
                <Button
                    key="Delete"
                    variant={impactedClusters.length === 0 ? 'primary' : 'danger'}
                    onClick={onDelete}
                    isDisabled={isRevokingBundle}
                >
                    Delete
                </Button>,
                <Button
                    key="Cancel"
                    variant="secondary"
                    onClick={onCancel}
                    isDisabled={isRevokingBundle}
                    ref={refCancelButton}
                >
                    Cancel
                </Button>,
            ]}
        >
            <div>
                <div className="pf-u-mb-md">
                    Bundle name: <strong>{bundleName}</strong>
                </div>
                {impactedClusters.length === 0 ? (
                    <Alert title="No clusters depend on this bundle" variant="info" isInline />
                ) : (
                    <>
                        {isRaceCondition ? (
                            <Alert
                                title="Additional clusters now depend on this bundle"
                                variant="warning"
                                isInline
                            >
                                You must confirm <strong>again</strong> to delete this bundle.
                            </Alert>
                        ) : (
                            <Alert title="Clusters depend on this bundle" variant="danger" isInline>
                                <p>
                                    In clusters that depend on this bundle, security deployments
                                    like Sensor will lose connectivity to Central.
                                </p>
                                <p className="pf-u-mt-md">
                                    We recommend that you <strong>replace</strong> this bundle in
                                    dependent clusters <strong>before</strong> you delete it.
                                </p>
                            </Alert>
                        )}
                        <h2 className="pf-u-mt-md">
                            <strong>Dependent clusters</strong>
                            <Badge className="pf-u-ml-md" isRead>
                                {impactedClusters.length}
                            </Badge>
                        </h2>
                        <List>
                            {impactedClusters.map(({ name }) => (
                                <ListItem key={name}>{name}</ListItem>
                            ))}
                        </List>
                    </>
                )}
                {alert}
            </div>
        </Modal>
    );
}

export default DeleteClusterInitBundleConfirmationModal;
