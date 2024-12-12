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
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';

import EmailTemplatePreview from '../components/EmailTemplatePreview';
import ReportParametersDetails from '../components/ReportParametersDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import exampleReportsCSVData from '../exampleReportsCSVData';
import { defaultEmailBody, getDefaultEmailSubject } from './emailTemplateFormUtils';

export type ReportReviewFormParams = {
    title: string;
    formValues: ReportFormValues;
};

const headingLevel = 'h3';

function ReportReviewForm({ title, formValues }: ReportReviewFormParams): ReactElement {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                variant="light"
                padding={{ default: 'noPadding' }}
                className="pf-v5-u-py-lg pf-v5-u-px-lg"
            >
                <ReportParametersDetails headingLevel={headingLevel} formValues={formValues} />
                <Divider component="div" className="pf-v5-u-py-md" />
                <NotifierConfigurationView
                    headingLevel={headingLevel}
                    customBodyDefault={defaultEmailBody}
                    customSubjectDefault={getDefaultEmailSubject(
                        formValues.reportParameters.reportName,
                        formValues.reportParameters.reportScope?.name
                    )}
                    notifierConfigurations={formValues.deliveryDestinations}
                    renderTemplatePreview={({
                        customBody,
                        customSubject,
                        customSubjectDefault,
                    }: TemplatePreviewArgs) => (
                        <EmailTemplatePreview
                            emailSubject={customSubject}
                            emailBody={customBody}
                            defaultEmailSubject={customSubjectDefault}
                            reportParameters={formValues.reportParameters}
                        />
                    )}
                />
                <Divider component="div" className="pf-v5-u-py-md" />
                <ScheduleDetails formValues={formValues} />
                <Divider component="div" className="pf-v5-u-py-md" />
                <Card>
                    <CardTitle>CVE report format</CardTitle>
                    <CardBody>
                        <TextContent>
                            <Text component={TextVariants.p}>
                                A sample preview to illustrate the selected parameters in a format
                                of CSV with nonactual data.
                            </Text>
                            <Text component={TextVariants.p} className="pf-v5-u-font-weight-bold">
                                Sara-reporting.csv
                            </Text>
                            <Text component={TextVariants.small}>
                                The data available in the preview is limited by the access scope of
                                your role
                            </Text>
                        </TextContent>
                        <div className="overflow-x-auto">
                            <Table>
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
                                        <Th>CVSS</Th>
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
                                            cvss,
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
                                                    <Td dataLabel="Severity">
                                                        <VulnerabilitySeverityIconText
                                                            severity={severity}
                                                        />
                                                    </Td>
                                                    <Td dataLabel="CVSS">{cvss}</Td>
                                                    <Td dataLabel="Discovered At">
                                                        {discoveredAt}
                                                    </Td>
                                                    <Td dataLabel="Reference">{reference}</Td>
                                                </Tr>
                                            );
                                        }
                                    )}
                                </Tbody>
                            </Table>
                        </div>
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default ReportReviewForm;
