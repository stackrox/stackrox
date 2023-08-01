import React, { ReactElement } from 'react';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import {
    Card,
    CardBody,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Text,
    TextContent,
    TextVariants,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import exampleReportsCSVData from '../exampleReportsCSVData';

import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';

export type ReportReviewFormParams = {
    title: string;
    formValues: ReportFormValues;
};

function ReportReviewForm({ title, formValues }: ReportReviewFormParams): ReactElement {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                variant="light"
                padding={{ default: 'noPadding' }}
                className="pf-u-py-lg pf-u-px-lg"
            >
                <ReportParametersDetails formValues={formValues} />
                <Divider component="div" className="pf-u-py-md" />
                <DeliveryDestinationsDetails formValues={formValues} />
                <Divider component="div" className="pf-u-py-md" />
                <ScheduleDetails formValues={formValues} />
                <Divider component="div" className="pf-u-py-md" />
                <Card>
                    <CardTitle>CVE report format</CardTitle>
                    <CardBody>
                        <TextContent>
                            <Text component={TextVariants.p}>
                                A sample preview to illustrate the selected parameters in a format
                                of CSV with nonactual data.
                            </Text>
                            <Text component={TextVariants.p} className="pf-u-font-weight-bold">
                                Sara-reporting.csv
                            </Text>
                            <Text component={TextVariants.small}>
                                The data available in the preview is limited by the access scope of
                                your role
                            </Text>
                        </TextContent>
                        <div className="overflow-x-auto">
                            <TableComposable>
                                <Thead noWrap>
                                    <Tr>
                                        <Th>Cluster</Th>
                                        <Th>Namespace</Th>
                                        <Th>Deployment</Th>
                                        <Th>Image</Th>
                                        <Th>Component</Th>
                                        <Th>CVE</Th>
                                        <Th>Fixable</Th>
                                        <Th>Component Upgrade</Th>
                                        <Th>Severity</Th>
                                        <Th>Discovered At</Th>
                                        <Th>Reference</Th>
                                    </Tr>
                                </Thead>
                                <Tbody>
                                    {exampleReportsCSVData.map(
                                        ({
                                            cluster,
                                            namespace,
                                            deployment,
                                            image,
                                            component,
                                            cve,
                                            fixable,
                                            componentUpgrade,
                                            severity,
                                            discoveredAt,
                                            reference,
                                        }) => {
                                            return (
                                                <Tr
                                                    key={`${cluster}/${namespace}/${deployment}/${image}/${component}/${cve}`}
                                                >
                                                    <Td dataLabel="Cluster">{cluster}</Td>
                                                    <Td dataLabel="Namespace">{namespace}</Td>
                                                    <Td dataLabel="Deployment">{deployment}</Td>
                                                    <Td dataLabel="Image">{image}</Td>
                                                    <Td dataLabel="Component">{component}</Td>
                                                    <Td dataLabel="CVE">{cve}</Td>
                                                    <Td dataLabel="Fixable">{fixable}</Td>
                                                    <Td dataLabel="Component Upgrade">
                                                        {componentUpgrade}
                                                    </Td>
                                                    <Td dataLabel="Severity">{severity}</Td>
                                                    <Td dataLabel="Discovered At">
                                                        {discoveredAt}
                                                    </Td>
                                                    <Td dataLabel="Reference">{reference}</Td>
                                                </Tr>
                                            );
                                        }
                                    )}
                                </Tbody>
                            </TableComposable>
                        </div>
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default ReportReviewForm;
