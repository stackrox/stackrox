import React, { ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core';

import { complianceEnhancedScanConfigsBasePath } from 'routePaths';

import { FormikProvider } from 'formik';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import ScanConfigOptions from './ScanConfigOptions';
import ClusterSelection from './ClusterSelection';
import ProfileSelection from './ProfileSelection';
import ScanConfigWizardFooter from './ScanConfigWizardFooter';
import useFormikScanConfig from './useFormikScanConfig';

const PARAMETERS = 'Set Parameters';
const PARAMETERS_ID = 'parameters';
const SELECT_CLUSTERS = 'Select clusters';
const SELECT_CLUSTERS_ID = 'clusters';
const SELECT_PROFILES = 'Select profiles';
const SELECT_PROFILES_ID = 'profiles';

function ScanConfigPage(): ReactElement {
    const history = useHistory();
    const formik = useFormikScanConfig();
    const { clusters, isLoading: isFetchingClusters } = useFetchClustersForPermissions([
        'Compliance',
    ]);

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
            component: <ProfileSelection />,
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
