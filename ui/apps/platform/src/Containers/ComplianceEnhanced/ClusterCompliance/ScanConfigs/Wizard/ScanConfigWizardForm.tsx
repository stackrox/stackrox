import React, { ReactElement, useCallback, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core';
import { FormikProvider } from 'formik';
import { complianceEnhancedScanConfigsPath } from 'routePaths';

import useRestQuery from 'hooks/useRestQuery';
import {
    saveScanConfig,
    listComplianceIntegrations,
    listComplianceProfiles,
} from 'services/ComplianceEnhancedService';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ScanConfigOptions from './ScanConfigOptions';
import ClusterSelection from './ClusterSelection';
import ProfileSelection from './ProfileSelection';
import ReviewConfig from './ReviewConfig';
import ScanConfigWizardFooter from './ScanConfigWizardFooter';
import useFormikScanConfig from './useFormikScanConfig';
import { convertFormikToScanConfig, ScanConfigFormValues } from '../compliance.scanConfigs.utils';

const PARAMETERS = 'Set Parameters';
const PARAMETERS_ID = 'parameters';
const SELECT_CLUSTERS = 'Select clusters';
const SELECT_CLUSTERS_ID = 'clusters';
const SELECT_PROFILES = 'Select profiles';
const SELECT_PROFILES_ID = 'profiles';
const REVIEW_CONFIG = 'Review and create';
const REVIEW_CONFIG_ID = 'review';

type ScanConfigWizardFormProps = {
    initialFormValues?: ScanConfigFormValues;
};

function ScanConfigWizardForm({ initialFormValues }: ScanConfigWizardFormProps): ReactElement {
    const history = useHistory();
    const formik = useFormikScanConfig(initialFormValues);
    const [isCreating, setIsCreating] = useState(false);
    const [createScanConfigError, setCreateScanConfigError] = useState('');

    const listClustersQuery = useCallback(() => listComplianceIntegrations(), []);
    const { data: clusters, loading: isFetchingClusters } = useRestQuery(listClustersQuery);

    const listProfilesQuery = useCallback(() => listComplianceProfiles(), []);
    const { data: profiles, loading: isFetchingProfiles } = useRestQuery(listProfilesQuery);

    async function onSave() {
        setIsCreating(true);
        setCreateScanConfigError('');
        const complianceScanConfig = convertFormikToScanConfig(formik.values);

        try {
            await saveScanConfig(complianceScanConfig);
            history.push(complianceEnhancedScanConfigsPath);
        } catch (error) {
            setCreateScanConfigError(getAxiosErrorMessage(error));
        } finally {
            setIsCreating(false);
        }
    }

    function onClose(): void {
        history.push(complianceEnhancedScanConfigsPath);
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
                <ClusterSelection
                    clusters={clusters || []}
                    isFetchingClusters={isFetchingClusters}
                />
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
                    clusters={clusters || []}
                    profiles={profiles || []}
                    errorMessage={createScanConfigError}
                />
            ),
            canJumpTo: Object.keys(formik.errors?.parameters || {}).length === 0,
        },
    ];

    const isEditing = initialFormValues?.id !== undefined;

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
                            onSave={onSave}
                            isSaving={isCreating}
                            proceedToNextStepIfValid={proceedToNextStepIfValid}
                            isEditing={isEditing}
                        />
                    }
                />
            </FormikProvider>
        </>
    );
}

export default ScanConfigWizardForm;
