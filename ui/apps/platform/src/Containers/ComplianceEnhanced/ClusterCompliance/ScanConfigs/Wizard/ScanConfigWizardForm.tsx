import React, { ReactElement, useCallback, useRef, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core';
import { FormikProvider } from 'formik';
import { complianceEnhancedScanConfigsPath } from 'routePaths';
import isEqual from 'lodash/isEqual';

import useRestQuery from 'hooks/useRestQuery';
import {
    saveScanConfig,
    listComplianceIntegrations,
    listComplianceSummaries,
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
    const [clustersUsedForProfileData, setClustersUsedForProfileData] = useState<string[]>([]);
    const alertRef = useRef<HTMLDivElement | null>(null);

    const listClustersQuery = useCallback(() => listComplianceIntegrations(), []);
    const { data: clusters, loading: isFetchingClusters } = useRestQuery(listClustersQuery);

    const listProfilesQuery = useCallback(() => {
        if (clustersUsedForProfileData.length > 0) {
            return listComplianceSummaries(clustersUsedForProfileData);
        }
        return Promise.resolve([]);
    }, [clustersUsedForProfileData]);
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

    function handleProfilesUpdate() {
        if (!isEqual(clustersUsedForProfileData, formik.values.clusters)) {
            setClustersUsedForProfileData(formik.values.clusters);
        }
    }

    function wizardStepChanged() {
        handleProfilesUpdate();
        setCreateScanConfigError('');
    }

    function scrollToAlert() {
        if (alertRef.current) {
            alertRef.current.scrollIntoView({
                behavior: 'smooth',
                block: 'start',
            });
        }
    }

    function onClose(): void {
        history.push(complianceEnhancedScanConfigsPath);
    }

    function setAllFieldsTouched(formikGroupKey: string): void {
        const groupHasNestedFields =
            typeof formik.values[formikGroupKey] === 'object' &&
            !Array.isArray(formik.values[formikGroupKey]);
        let touchedState;

        if (groupHasNestedFields) {
            touchedState = Object.keys(formik.values[formikGroupKey]).reduce((acc, field) => {
                acc[field] = true;
                return acc;
            }, {});
            formik.setTouched({ ...formik.touched, [formikGroupKey]: touchedState });
        } else {
            formik.setTouched({ ...formik.touched, [formikGroupKey]: true });
        }
    }

    function proceedToNextStepIfValid(
        navigateToNextStep: () => void,
        formikGroupKey: string
    ): void {
        const hasNoErrors = Object.keys(formik.errors?.[formikGroupKey] || {}).length === 0;
        if (hasNoErrors) {
            navigateToNextStep();
        } else {
            setAllFieldsTouched(formikGroupKey);
            scrollToAlert();
        }
    }

    function canJumpToSelectClusters() {
        return Object.keys(formik.errors?.parameters || {}).length === 0;
    }

    function canJumpToSelectProfiles() {
        return canJumpToSelectClusters() && Object.keys(formik.errors?.clusters || {}).length === 0;
    }

    function canJumpToReviewConfig() {
        return canJumpToSelectProfiles() && Object.keys(formik.errors?.profiles || {}).length === 0;
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
                    alertRef={alertRef}
                    clusters={clusters || []}
                    isFetchingClusters={isFetchingClusters}
                />
            ),
            canJumpTo: canJumpToSelectClusters(),
        },
        {
            name: SELECT_PROFILES,
            id: SELECT_PROFILES_ID,
            component: (
                <ProfileSelection
                    alertRef={alertRef}
                    profiles={profiles || []}
                    isFetchingProfiles={isFetchingProfiles}
                />
            ),
            canJumpTo: canJumpToSelectProfiles(),
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
            canJumpTo: canJumpToReviewConfig(),
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
                    onCurrentStepChanged={wizardStepChanged}
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
