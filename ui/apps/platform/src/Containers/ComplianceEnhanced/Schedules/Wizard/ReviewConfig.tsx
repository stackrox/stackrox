import React from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import {
    Alert,
    Badge,
    Divider,
    Flex,
    FlexItem,
    List,
    ListItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';
import { ComplianceIntegration } from 'services/ComplianceIntegrationService';

import {
    convertFormikParametersToSchedule,
    getBodyDefault,
    getSubjectDefault,
    ScanConfigFormValues,
} from '../compliance.scanConfigs.utils';
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
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
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
                spaceItems={{ default: 'spaceItemsLg' }}
                className="pf-v5-u-pt-lg pf-v5-u-px-lg"
            >
                <ScanConfigParametersView
                    headingLevel={headingLevel}
                    scanName={formikValues.parameters.name}
                    description={formikValues.parameters.description}
                    scanSchedule={scanSchedule}
                />
                <Flex direction={{ default: 'column' }}>
                    <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                        <Title headingLevel={headingLevel}>Clusters</Title>
                        <Badge isRead>{selectedClusters.length}</Badge>
                    </Flex>
                    <List isPlain>
                        {selectedClusters.map((cluster) => (
                            <ListItem key={cluster.id}>{cluster.clusterName}</ListItem>
                        ))}
                    </List>
                </Flex>
                <ScanConfigProfilesView
                    headingLevel={headingLevel}
                    profiles={formikValues.profiles}
                />
                <NotifierConfigurationView
                    headingLevel={headingLevel}
                    customBodyDefault={getBodyDefault(formikValues.profiles)}
                    customSubjectDefault={getSubjectDefault(
                        formikValues.parameters.name,
                        formikValues.profiles
                    )}
                    notifierConfigurations={formikValues.report.notifierConfigurations}
                />
                <Alert
                    variant="info"
                    title="Save for new versus existing scan schedule"
                    component="p"
                    isInline
                >
                    Compliance Operator runs a new scan schedule immediately upon creation, but does
                    not run until scheduled time when you save changes to an existing scan schedule.
                </Alert>
            </Flex>
        </>
    );
}

export default ReviewConfig;
