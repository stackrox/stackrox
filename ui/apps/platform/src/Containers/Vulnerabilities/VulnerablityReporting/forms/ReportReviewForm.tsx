import React, { ReactElement } from 'react';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
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

import { getDate } from 'utils/dateUtils';
import {
    cvesDiscoveredSinceLabelMap,
    imageTypeLabelMap,
} from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import { fixabilityLabels } from 'constants/reportConstants';

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';

import exampleReportsCSVData from '../exampleReportsCSVData';

export type ReportReviewFormParams = {
    formValues: ReportFormValues;
};

function ReportReviewForm({ formValues }: ReportReviewFormParams): ReactElement {
    const cveSeverities =
        formValues.reportParameters.cveSeverities.length !== 0 ? (
            formValues.reportParameters.cveSeverities.map((severity) => (
                <li key={severity}>
                    <VulnerabilitySeverityIconText severity={severity} />
                </li>
            ))
        ) : (
            <li>None</li>
        );
    const cveStatuses =
        formValues.reportParameters.cveStatus.length !== 0 ? (
            formValues.reportParameters.cveStatus.map((status) => (
                <li key={status}>{fixabilityLabels[status]}</li>
            ))
        ) : (
            <li>None</li>
        );
    const imageTypes =
        formValues.reportParameters.imageType.length !== 0 ? (
            formValues.reportParameters.imageType.map((type) => (
                <li key={type}>{imageTypeLabelMap[type]}</li>
            ))
        ) : (
            <li>None</li>
        );

    const deliveryDestinations =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => (
                <li key={deliveryDestination.notifier?.id}>{deliveryDestination.notifier?.name}</li>
            ))
        ) : (
            <li>None</li>
        );

    const mailingLists =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => {
                const emails = deliveryDestination.mailingLists.join(', ');
                return <li key={emails}>{emails}</li>;
            })
        ) : (
            <li>None</li>
        );

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Review and create</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                variant="light"
                padding={{ default: 'noPadding' }}
                className="pf-u-py-lg pf-u-px-lg"
            >
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <Title headingLevel="h3">Report parameters</Title>
                    </FlexItem>
                    <FlexItem flex={{ default: 'flexNone' }}>
                        <DescriptionList
                            columnModifier={{
                                default: '2Col',
                                md: '2Col',
                                sm: '1Col',
                            }}
                        >
                            <DescriptionListGroup>
                                <DescriptionListTerm>Report name</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {formValues.reportParameters.reportName || 'None'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Description</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {formValues.reportParameters.description || 'None'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>CVE severity</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ul>{cveSeverities}</ul>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>CVE status</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ul>{cveStatuses}</ul>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Report scope</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {formValues.reportParameters.reportScope?.name || 'None'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Image type</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ul>{imageTypes}</ul>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>CVEs discovered since</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {formValues.reportParameters.cvesDiscoveredSince ===
                                        'START_DATE' &&
                                    !!formValues.reportParameters.cvesDiscoveredStartDate
                                        ? getDate(
                                              formValues.reportParameters.cvesDiscoveredStartDate
                                          )
                                        : cvesDiscoveredSinceLabelMap[
                                              formValues.reportParameters.cvesDiscoveredSince
                                          ]}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </FlexItem>
                </Flex>
                <Divider component="div" className="pf-u-py-md" />
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <Title headingLevel="h3">Delivery destinations</Title>
                    </FlexItem>
                    <FlexItem flex={{ default: 'flexNone' }}>
                        <DescriptionList
                            columnModifier={{
                                default: '2Col',
                            }}
                        >
                            <DescriptionListGroup>
                                <DescriptionListTerm>Email notifier</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ul>{deliveryDestinations}</ul>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Destribution list</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ul>{mailingLists}</ul>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </FlexItem>
                </Flex>
                <Divider component="div" className="pf-u-py-md" />
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <Title headingLevel="h3">Schedule details</Title>
                    </FlexItem>
                    <FlexItem flex={{ default: 'flexNone' }}>
                        <TextContent>
                            <Text component={TextVariants.p}>
                                Report is scheduled to be sent on Monday every week
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
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
