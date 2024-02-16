import React from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import {
    Alert,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Stack,
    StackItem,
    Text,
    TextList,
    TextListItem,
    TextVariants,
    Title,
} from '@patternfly/react-core';

import {
    ComplianceProfileSummary,
    ComplianceIntegration,
} from 'services/ComplianceEnhancedService';

import {
    convertFormikParametersToSchedule,
    formatScanSchedule,
    ScanConfigFormValues,
} from '../compliance.scanConfigs.utils';

export type ProfileSelectionProps = {
    clusters: ComplianceIntegration[];
    profiles: ComplianceProfileSummary[];
    errorMessage: string;
};

function ReviewConfig({ clusters, profiles, errorMessage }: ProfileSelectionProps) {
    const { values: formikValues }: FormikContextType<ScanConfigFormValues> = useFormikContext();

    const scanSchedule = convertFormikParametersToSchedule(formikValues.parameters);
    const formattedScanSchedule = formatScanSchedule(scanSchedule);

    function findById<T, K extends keyof T>(selectedIds: string[], items: T[], idKey: K): T[] {
        return selectedIds
            .map((id) => items.find((item) => String(item[idKey]) === id))
            .filter((item): item is T => item !== undefined);
    }

    const selectedClusters = findById(formikValues.clusters, clusters, 'clusterId');
    const selectedProfiles = findById(formikValues.profiles, profiles, 'name');

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Review</Title>
                    </FlexItem>
                    <FlexItem>Review and create your scan configuration</FlexItem>
                    {errorMessage && (
                        <Alert
                            title={'Scan configuration request failure'}
                            variant="danger"
                            isInline
                        >
                            {errorMessage}
                        </Alert>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Stack hasGutter className="pf-u-py-lg pf-u-px-lg">
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Name
                    </Text>
                    <Text>{formikValues.parameters.name}</Text>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Schedule
                    </Text>
                    <Text>{formattedScanSchedule}</Text>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Clusters
                    </Text>
                    <TextList isPlain>
                        {selectedClusters.map((cluster) => (
                            <TextListItem key={cluster.id}>{cluster.clusterName}</TextListItem>
                        ))}
                    </TextList>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Profiles
                    </Text>
                    <TextList isPlain>
                        {selectedProfiles.map((profile) => (
                            <TextListItem key={profile.name}>{profile.name}</TextListItem>
                        ))}
                    </TextList>
                </StackItem>
            </Stack>
        </>
    );
}

export default ReviewConfig;
