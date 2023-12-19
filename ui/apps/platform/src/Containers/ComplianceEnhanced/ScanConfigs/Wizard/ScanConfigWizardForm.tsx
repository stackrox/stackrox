import React, { ReactElement, useCallback, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core';
import { FormikProvider } from 'formik';
import { complianceEnhancedScanConfigsBasePath } from 'routePaths';

import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import useRestQuery from 'hooks/useRestQuery';
import { createScanConfig, listComplianceProfiles } from 'services/ComplianceEnhancedService';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ScanConfigOptions from './ScanConfigOptions';
import ClusterSelection from './ClusterSelection';
import ProfileSelection from './ProfileSelection';
import ReviewConfig from './ReviewConfig';
import ScanConfigWizardFooter from './ScanConfigWizardFooter';
import useFormikScanConfig from './useFormikScanConfig';
import { convertFormikToScanConfig } from '../compliance.scanConfigs.utils';

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
    const [isCreating, setIsCreating] = useState(false);
    const [createScanConfigError, setCreateScanConfigError] = useState('');

    const listQuery = useCallback(() => listComplianceProfiles(), []);
    const { data: profiles, loading: isFetchingProfiles } = useRestQuery(listQuery);

    async function onCreate() {
        setIsCreating(true);
        setCreateScanConfigError('');
        const complianceScanConfig = convertFormikToScanConfig(formik.values);

        try {
            await createScanConfig(complianceScanConfig);
            history.push(complianceEnhancedScanConfigsBasePath);
        } catch (error) {
            setCreateScanConfigError(getAxiosErrorMessage(error));
        } finally {
            setIsCreating(false);
        }
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
            component: (
                <ReviewConfig
                    clusters={clusters}
                    profiles={profiles || []}
                    errorMessage={createScanConfigError}
                />
            ),
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
                            isSaving={isCreating}
                            proceedToNextStepIfValid={proceedToNextStepIfValid}
                        />
                    }
                />
            </FormikProvider>
        </>
    );
}

export default ScanConfigPage;
