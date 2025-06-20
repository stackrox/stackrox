import React, { ReactElement, useCallback, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Modal } from '@patternfly/react-core';
import {
    Wizard,
    WizardContextConsumer,
    WizardFooter,
    WizardStep,
} from '@patternfly/react-core/deprecated';
import { FormikProvider } from 'formik';
import { complianceEnhancedSchedulesPath } from 'routePaths';
import isEqual from 'lodash/isEqual';

import useAnalytics, {
    COMPLIANCE_SCHEDULES_WIZARD_SAVE_CLICKED,
    COMPLIANCE_SCHEDULES_WIZARD_STEP_CHANGED,
} from 'hooks/useAnalytics';
import useModal from 'hooks/useModal';
import useRestQuery from 'hooks/useRestQuery';
import { saveScanConfig } from 'services/ComplianceScanConfigurationService';
import { listComplianceIntegrations } from 'services/ComplianceIntegrationService';
import { listProfileSummaries } from 'services/ComplianceProfileService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ScanConfigOptions from './ScanConfigOptions';
import ClusterSelection from './ClusterSelection';
import ProfileSelection from './ProfileSelection';
import ReportConfiguration from './ReportConfiguration';
import ReviewConfig from './ReviewConfig';
import useFormikScanConfig from './useFormikScanConfig';
import { convertFormikToScanConfig, ScanConfigFormValues } from '../compliance.scanConfigs.utils';

const PARAMETERS = 'Set parameters';
const PARAMETERS_ID = 'parameters';
const SELECT_CLUSTERS = 'Select clusters';
const SELECT_CLUSTERS_ID = 'clusters';
const SELECT_PROFILES = 'Select profiles';
const SELECT_PROFILES_ID = 'profiles';
const CONFIGURE_REPORT = 'Configure report';
const CONFIGURE_REPORT_ID = 'report';
const REVIEW_CONFIG = 'Review';
const REVIEW_CONFIG_ID = 'review';

type ScanConfigWizardFormProps = {
    initialFormValues?: ScanConfigFormValues;
};

function ScanConfigWizardForm({ initialFormValues }: ScanConfigWizardFormProps): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();
    const formik = useFormikScanConfig(initialFormValues);
    const [isCreating, setIsCreating] = useState(false);
    const [createScanConfigError, setCreateScanConfigError] = useState('');
    const [clustersUsedForProfileData, setClustersUsedForProfileData] = useState<string[]>([]);
    const alertRef = useRef<HTMLDivElement | null>(null);

    const listClustersQuery = useCallback(() => listComplianceIntegrations(), []);
    const { data: clusters, isLoading: isFetchingClusters } = useRestQuery(listClustersQuery);

    const listProfilesQuery = useCallback(() => {
        if (clustersUsedForProfileData.length > 0) {
            return listProfileSummaries(clustersUsedForProfileData);
        }
        return Promise.resolve([]);
    }, [clustersUsedForProfileData]);
    const { data: profiles, isLoading: isFetchingProfiles } = useRestQuery(listProfilesQuery);

    const { isModalOpen, openModal, closeModal } = useModal();

    async function onSave() {
        setIsCreating(true);
        setCreateScanConfigError('');
        const complianceScanConfig = convertFormikToScanConfig(formik.values);

        try {
            await saveScanConfig(complianceScanConfig);
            analyticsTrack({
                event: COMPLIANCE_SCHEDULES_WIZARD_SAVE_CLICKED,
                properties: {
                    success: true,
                    errorMessage: '',
                },
            });
            navigate(complianceEnhancedSchedulesPath);
        } catch (error) {
            analyticsTrack({
                event: COMPLIANCE_SCHEDULES_WIZARD_SAVE_CLICKED,
                properties: {
                    success: false,
                    errorMessage: getAxiosErrorMessage(error),
                },
            });
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

    function wizardStepChanged(step: WizardStep) {
        if (typeof step.id === 'string') {
            analyticsTrack({
                event: COMPLIANCE_SCHEDULES_WIZARD_STEP_CHANGED,
                properties: {
                    step: step.id,
                },
            });
        }
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
        navigate(complianceEnhancedSchedulesPath);
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

    function canJumpToConfigureReport() {
        return canJumpToSelectProfiles() && Object.keys(formik.errors?.profiles || {}).length === 0;
    }

    function canJumpToReviewConfig() {
        return canJumpToConfigureReport() && Object.keys(formik.errors?.report || {}).length === 0;
    }

    function allClustersAreUnhealthy(clusters): boolean {
        return clusters?.every((cluster) => cluster.status === 'UNHEALTHY') as boolean;
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
            name: CONFIGURE_REPORT,
            id: CONFIGURE_REPORT_ID,
            component: <ReportConfiguration />,
            canJumpTo: canJumpToConfigureReport(),
        },
        {
            name: REVIEW_CONFIG,
            id: REVIEW_CONFIG_ID,
            component: (
                <ReviewConfig clusters={clusters || []} errorMessage={createScanConfigError} />
            ),
            canJumpTo: canJumpToReviewConfig(),
        },
    ];

    const firstStepId = wizardSteps[0].id;
    const lastStepId = wizardSteps[wizardSteps.length - 1].id;

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
                        <WizardFooter>
                            <WizardContextConsumer>
                                {({ activeStep, onNext, onBack, onClose }) => (
                                    <>
                                        {activeStep.id !== lastStepId ? (
                                            <Button
                                                variant="primary"
                                                type="submit"
                                                isDisabled={
                                                    allClustersAreUnhealthy(clusters) &&
                                                    !initialFormValues &&
                                                    activeStep.id === wizardSteps[1].id
                                                }
                                                onClick={() =>
                                                    proceedToNextStepIfValid(
                                                        onNext,
                                                        String(activeStep.id)
                                                    )
                                                }
                                            >
                                                Next
                                            </Button>
                                        ) : (
                                            <Button
                                                variant="primary"
                                                type="submit"
                                                isDisabled={isCreating}
                                                onClick={onSave}
                                                isLoading={isCreating}
                                            >
                                                Save
                                            </Button>
                                        )}
                                        <Button
                                            variant="secondary"
                                            onClick={onBack}
                                            isDisabled={activeStep.id === firstStepId}
                                        >
                                            Back
                                        </Button>
                                        <Button variant="link" onClick={openModal}>
                                            Cancel
                                        </Button>
                                        <Modal
                                            variant="small"
                                            title="Confirm cancel"
                                            isOpen={isModalOpen}
                                            onClose={closeModal}
                                            actions={[
                                                <Button
                                                    key="confirm"
                                                    variant="primary"
                                                    onClick={onClose}
                                                >
                                                    Confirm
                                                </Button>,
                                                <Button
                                                    key="cancel"
                                                    variant="secondary"
                                                    onClick={closeModal}
                                                >
                                                    Cancel
                                                </Button>,
                                            ]}
                                        >
                                            <p>
                                                Are you sure you want to cancel? Any unsaved changes
                                                will be lost. You will be taken back to the list of
                                                scan configurations.
                                            </p>
                                        </Modal>
                                    </>
                                )}
                            </WizardContextConsumer>
                        </WizardFooter>
                    }
                />
            </FormikProvider>
        </>
    );
}

export default ScanConfigWizardForm;
