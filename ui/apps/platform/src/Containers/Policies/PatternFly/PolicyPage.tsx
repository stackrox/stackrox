import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Bullseye, PageSection, Spinner } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import { fetchClustersAsArray } from 'services/ClustersService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { getPolicy, updatePolicyDisabledState } from 'services/PoliciesService';
import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { Policy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import { getClientWizardPolicy } from './policies.utils';
import PolicyDetail from './Detail/PolicyDetail';
import PolicyWizard from './Wizard/PolicyWizard';

function clonePolicy(policy: Policy) {
    /*
     * Default policies will have the "criteriaLocked" and "mitreVectorsLocked" fields set to true.
     * When we clone these policies, we'll need to set them to false to allow users to edit
     * both the policy criteria and mitre attack vectors
     */
    return {
        ...policy,
        criteriaLocked: false,
        id: '',
        isDefault: false,
        mitreVectorsLocked: false,
        name: `${policy.name} (COPY)`,
    };
}

const initialPolicy: Policy = {
    id: '',
    name: '',
    description: '',
    severity: 'LOW_SEVERITY',
    disabled: false,
    lifecycleStages: [],
    notifiers: [],
    lastUpdated: null,
    eventSource: 'NOT_APPLICABLE',
    isDefault: false,
    rationale: '',
    remediation: '',
    categories: [],
    fields: null,
    exclusions: [],
    scope: [],
    enforcementActions: [],
    excludedImageNames: [],
    excludedDeploymentScopes: [],
    SORT_name: '', // For internal use only.
    SORT_lifecycleStage: '', // For internal use only.
    SORT_enforcement: false, // For internal use only.
    policyVersion: '',
    policySections: [],
    mitreAttackVectors: [],
    criteriaLocked: false,
    mitreVectorsLocked: false,
};

type PolicyPageProps = {
    hasWriteAccessForPolicy: boolean;
    pageAction?: ExtendedPageAction;
    policyId?: string;
};

function PolicyPage({
    hasWriteAccessForPolicy,
    pageAction,
    policyId,
}: PolicyPageProps): ReactElement {
    const [clusters, setClusters] = useState<Cluster[]>([]);

    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);

    const [policy, setPolicy] = useState<Policy>(initialPolicy);
    const [policyError, setPolicyError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        fetchClustersAsArray()
            .then((data) => {
                setClusters(data as Cluster[]);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    useEffect(() => {
        fetchNotifierIntegrations()
            .then((data) => {
                setNotifiers(data as NotifierIntegration[]);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    useEffect(() => {
        setPolicyError(null);
        if (policyId) {
            // action is 'clone' or 'edit' or undefined
            setIsLoading(true);
            getPolicy(policyId)
                .then((data) => {
                    const clientWizardPolicy = getClientWizardPolicy(data);
                    setPolicy(
                        pageAction === 'clone'
                            ? clonePolicy(clientWizardPolicy)
                            : clientWizardPolicy
                    );
                })
                .catch((error) => {
                    setPolicy(initialPolicy);
                    setPolicyError(
                        <Alert title="Request failure for policy" variant="danger" isInline>
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        } else {
            // action is 'create'
            setPolicy(initialPolicy);
        }
    }, [pageAction, policyId]);

    function handleUpdateDisabledState(id: string, disabled: boolean) {
        return updatePolicyDisabledState(id, disabled).then(() => {
            /*
             * If success, render PolicyDetail element with updated policy.
             * If failure, PolicyDetail element has catch block to display error.
             */
            return getPolicy(id).then((data) => {
                setPolicy(data);
            });
        });
    }

    return (
        <>
            <PageTitle title="Policies - Policy" />
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : (
                policyError || // TODO ROX-8487: Improve PolicyPage when request fails
                (pageAction ? (
                    <PolicyWizard
                        pageAction={pageAction}
                        policy={policy}
                        clusters={clusters}
                        notifiers={notifiers}
                    />
                ) : (
                    <PolicyDetail
                        clusters={clusters}
                        handleUpdateDisabledState={handleUpdateDisabledState}
                        hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                        notifiers={notifiers}
                        policy={policy}
                    />
                ))
            )}
        </>
    );
}

export default PolicyPage;
