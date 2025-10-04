import React, { useCallback, useRef, useState } from 'react';
import type { ReactElement, RefObject } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import {
    Button,
    Modal,
    Wizard,
    WizardFooter,
    WizardStep,
    useWizardContext,
} from '@patternfly/react-core';
import type { WizardStepType } from '@patternfly/react-core';
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
import { convertFormikToScanConfig } from '../compliance.scanConfigs.utils';
import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

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

type CustomWizardFooterProps = {
    stepId: string;
    formik: ReturnType<typeof useFormikScanConfig>;
    alertRef: RefObject<HTMLDivElement>;
    openModal: () => void;
    validate?: () => boolean;
};

function CustomWizardFooter({
    stepId,
    formik,
    alertRef,
    openModal,
    validate = () => true,
}: CustomWizardFooterProps) {
    const { activeStep, goToNextStep, goToPrevStep } = useWizardContext();

    function scrollToAlert() {
        if (alertRef.current) {
            alertRef.current.scrollIntoView({
                behavior: 'smooth',
                block: 'start',
            });
        }
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

    function handleNext() {
        const hasNoErrors = Object.keys(formik.errors?.[stepId] || {}).length === 0;

        if (!hasNoErrors) {
            setAllFieldsTouched(stepId);
            scrollToAlert();
            return; // Don't navigate if validation fails
        }

        // Additional validation check if provided
        if (!validate()) {
            return;
        }

        // If validation passes, navigate to next step
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        goToNextStep();
    }

    return (
        <WizardFooter
            activeStep={activeStep}
            isBackDisabled={activeStep.name === PARAMETERS}
            onNext={handleNext}
            onBack={goToPrevStep}
            onClose={openModal}
        />
    );
}

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

    function wizardStepChanged(_event: unknown, currentStep: WizardStepType): void {
        if (currentStep?.id) {
            analyticsTrack({
                event: COMPLIANCE_SCHEDULES_WIZARD_STEP_CHANGED,
                properties: {
                    step: String(currentStep.id),
                },
            });
        }

        handleProfilesUpdate();
        setCreateScanConfigError('');
    }

    function onClose(): void {
        navigate(complianceEnhancedSchedulesPath);
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

    function allClustersAreUnhealthy(): boolean {
        return clusters?.every((cluster) => cluster.status === 'UNHEALTHY') || false;
    }

    return (
        <>
            <FormikProvider value={formik}>
                <Wizard
                    navAriaLabel="Scan schedule configuration steps"
                    onSave={onSave}
                    onStepChange={wizardStepChanged}
                >
                    <WizardStep
                        name={PARAMETERS}
                        id={PARAMETERS_ID}
                        key={PARAMETERS_ID}
                        body={{ hasNoPadding: true }}
                        footer={
                            <CustomWizardFooter
                                stepId={PARAMETERS_ID}
                                formik={formik}
                                alertRef={alertRef}
                                openModal={openModal}
                            />
                        }
                    >
                        <ScanConfigOptions />
                    </WizardStep>
                    <WizardStep
                        name={SELECT_CLUSTERS}
                        id={SELECT_CLUSTERS_ID}
                        key={SELECT_CLUSTERS_ID}
                        body={{ hasNoPadding: true }}
                        isDisabled={!canJumpToSelectClusters()}
                        footer={
                            <CustomWizardFooter
                                stepId={SELECT_CLUSTERS_ID}
                                formik={formik}
                                alertRef={alertRef}
                                openModal={openModal}
                                validate={() => !(allClustersAreUnhealthy() && !initialFormValues)}
                            />
                        }
                    >
                        <ClusterSelection
                            alertRef={alertRef}
                            clusters={clusters || []}
                            isFetchingClusters={isFetchingClusters}
                        />
                    </WizardStep>
                    <WizardStep
                        name={SELECT_PROFILES}
                        id={SELECT_PROFILES_ID}
                        key={SELECT_PROFILES_ID}
                        body={{ hasNoPadding: true }}
                        isDisabled={!canJumpToSelectProfiles()}
                        footer={
                            <CustomWizardFooter
                                stepId={SELECT_PROFILES_ID}
                                formik={formik}
                                alertRef={alertRef}
                                openModal={openModal}
                            />
                        }
                    >
                        <ProfileSelection
                            alertRef={alertRef}
                            profiles={profiles || []}
                            isFetchingProfiles={isFetchingProfiles}
                        />
                    </WizardStep>
                    <WizardStep
                        name={CONFIGURE_REPORT}
                        id={CONFIGURE_REPORT_ID}
                        key={CONFIGURE_REPORT_ID}
                        body={{ hasNoPadding: true }}
                        isDisabled={!canJumpToConfigureReport()}
                        footer={
                            <CustomWizardFooter
                                stepId={CONFIGURE_REPORT_ID}
                                formik={formik}
                                alertRef={alertRef}
                                openModal={openModal}
                            />
                        }
                    >
                        <ReportConfiguration />
                    </WizardStep>
                    <WizardStep
                        name={REVIEW_CONFIG}
                        id={REVIEW_CONFIG_ID}
                        key={REVIEW_CONFIG_ID}
                        body={{ hasNoPadding: true }}
                        isDisabled={!canJumpToReviewConfig()}
                        footer={{
                            nextButtonProps: { isLoading: isCreating },
                            nextButtonText: 'Save',
                            onClose: openModal,
                        }}
                    >
                        <ReviewConfig
                            clusters={clusters || []}
                            errorMessage={createScanConfigError}
                        />
                    </WizardStep>
                </Wizard>
            </FormikProvider>
            <Modal
                variant="small"
                title="Confirm cancel"
                isOpen={isModalOpen}
                onClose={closeModal}
                actions={[
                    <Button key="confirm" variant="primary" onClick={onClose}>
                        Confirm
                    </Button>,
                    <Button key="cancel" variant="secondary" onClick={closeModal}>
                        Cancel
                    </Button>,
                ]}
            >
                <p>
                    Are you sure you want to cancel? Any unsaved changes will be lost. You will be
                    taken back to the list of scan configurations.
                </p>
            </Modal>
        </>
    );
}

export default ScanConfigWizardForm;
