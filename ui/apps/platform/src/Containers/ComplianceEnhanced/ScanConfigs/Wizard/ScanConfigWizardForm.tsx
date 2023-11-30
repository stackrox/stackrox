import React, { ReactElement, useCallback } from 'react';
import { useHistory } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core';
import { FormikProvider } from 'formik';
import { complianceEnhancedScanConfigsBasePath } from 'routePaths';

import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import useRestQuery from 'hooks/useRestQuery';
import { listComplianceProfiles } from 'services/ComplianceEnhancedService';

import ScanConfigOptions from './ScanConfigOptions';
import ClusterSelection from './ClusterSelection';
import ProfileSelection from './ProfileSelection';
import ReviewConfig from './ReviewConfig';
import ScanConfigWizardFooter from './ScanConfigWizardFooter';
import useFormikScanConfig from './useFormikScanConfig';

const PARAMETERS = 'Set Parameters';
const PARAMETERS_ID = 'parameters';
const SELECT_CLUSTERS = 'Select clusters';
const SELECT_CLUSTERS_ID = 'clusters';
const SELECT_PROFILES = 'Select profiles';
const SELECT_PROFILES_ID = 'profiles';
const REVIEW_CONFIG = 'Review and create';
const REVIEW_CONFIG_ID = 'review';

function ScanConfigPage(): ReactElement {
    const history = useHistory();
    const formik = useFormikScanConfig();
    const { clusters, isLoading: isFetchingClusters } = useFetchClustersForPermissions([
        'Compliance',
    ]);

    const listQuery = useCallback(() => listComplianceProfiles(), []);
    const { data: profiles, loading: isFetchingProfiles } = useRestQuery(listQuery);

    function onCreate() {
        // TODO: create scan
    }

    function onClose(): void {
        history.push(complianceEnhancedScanConfigsBasePath);
    }

    function setAllFieldsTouched(formikGroupKey: string): void {
        const fields = Object.keys(formik.values[formikGroupKey]);
        const touchedState = fields.reduce((acc, field) => ({ ...acc, [field]: true }), {});
        formik.setTouched({ [formikGroupKey]: touchedState });
    }

    function proceedToNextStepIfValid(
        navigateToNextStep: () => void,
        formikGroupKey: string
    ): void {
        if (Object.keys(formik.errors?.[formikGroupKey] || {}).length === 0) {
            navigateToNextStep();
        } else {
            setAllFieldsTouched(formikGroupKey);
        }
    }

    const wizardSteps: WizardStep[] = [
        {
            name: PARAMETERS,
            id: PARAMETERS_ID,
            component: <ScanConfigOptions />,
        },
        {
            name: SELECT_CLUSTERS,
            id: SELECT_CLUSTERS_ID,
            component: (
                <ClusterSelection clusters={clusters} isFetchingClusters={isFetchingClusters} />
            ),
            canJumpTo: Object.keys(formik.errors?.parameters || {}).length === 0,
        },
        {
            name: SELECT_PROFILES,
            id: SELECT_PROFILES_ID,
            component: (
                <ProfileSelection
                    profiles={profiles || []}
                    isFetchingProfiles={isFetchingProfiles}
                />
            ),
            canJumpTo: Object.keys(formik.errors?.parameters || {}).length === 0,
        },
        {
            name: REVIEW_CONFIG,
            id: REVIEW_CONFIG_ID,
            component: <ReviewConfig />,
            canJumpTo: Object.keys(formik.errors?.parameters || {}).length === 0,
        },
    ];

    return (
        <>
            <FormikProvider value={formik}>
                <Wizard
                    navAriaLabel="Scan configuration creation steps"
                    mainAriaLabel="Scan configuration creation content"
                    hasNoBodyPadding
                    steps={wizardSteps}
                    onClose={onClose}
                    footer={
                        <ScanConfigWizardFooter
                            wizardSteps={wizardSteps}
                            onSave={onCreate}
                            isSaving={false}
                            proceedToNextStepIfValid={proceedToNextStepIfValid}
                        />
                    }
                />
            </FormikProvider>
        </>
    );
}

export default ScanConfigPage;
