import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import { Alert, Divider, Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';

import type { ComplianceIntegration } from 'services/ComplianceIntegrationService';

import { convertFormikParametersToSchedule } from '../compliance.scanConfigs.utils';
import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';
import ScanConfigClustersView from '../components/ScanConfigClustersView';
import ScanConfigDeliveryView from '../components/ScanConfigDeliveryView';
import ScanConfigParametersView from '../components/ScanConfigParametersView';
import ScanConfigProfilesView from '../components/ScanConfigProfilesView';

const headingLevel = 'h3';

export type ReviewConfigProps = {
    clusters: ComplianceIntegration[];
    errorMessage: string;
};

function ReviewConfig({ clusters, errorMessage }: ReviewConfigProps) {
    const { values: formikValues }: FormikContextType<ScanConfigFormValues> = useFormikContext();

    const scanSchedule = convertFormikParametersToSchedule(formikValues.parameters);

    function findById<T, K extends keyof T>(selectedIds: string[], items: T[], idKey: K): T[] {
        return selectedIds
            .map((id) => items.find((item) => String(item[idKey]) === id))
            .filter((item): item is T => item !== undefined);
    }

    const selectedClusters = findById(formikValues.clusters, clusters, 'clusterId');

    return (
        <>
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v6-u-py-lg pf-v6-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Review</Title>
                    </FlexItem>
                    <FlexItem>Review the scan schedule before you save changes</FlexItem>
                    {errorMessage && (
                        <Alert
                            title={'Scan configuration request failure'}
                            component="p"
                            variant="danger"
                            isInline
                        >
                            {errorMessage}
                        </Alert>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsMd' }}
                className="pf-v6-u-pt-lg pf-v6-u-px-lg"
            >
                <ScanConfigParametersView
                    headingLevel={headingLevel}
                    scanName={formikValues.parameters.name}
                    description={formikValues.parameters.description}
                    scanSchedule={scanSchedule}
                />
                <ScanConfigClustersView clusters={selectedClusters} headingLevel={headingLevel} />
                <ScanConfigProfilesView
                    headingLevel={headingLevel}
                    profiles={formikValues.profiles}
                />
                <ScanConfigDeliveryView
                    headingLevel={headingLevel}
                    notifiers={formikValues.report.notifierConfigurations}
                />
                <Alert
                    variant="info"
                    title="Save for new versus existing scan schedule"
                    component="p"
                    isInline
                    className="pf-v6-u-mb-lg"
                >
                    Compliance Operator runs a new scan schedule immediately upon creation, but does
                    not run until scheduled time when you save changes to an existing scan schedule.
                </Alert>
            </Flex>
        </>
    );
}

export default ReviewConfig;
