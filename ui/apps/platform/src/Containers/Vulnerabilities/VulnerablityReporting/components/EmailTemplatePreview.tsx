import {
    Card,
    CardBody,
    CardFooter,
    CardTitle,
    Flex,
    FlexItem,
    Text,
    TextContent,
    TextVariants,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import React, { useState } from 'react';

import { vulnerabilitySeverityLabels } from 'messages/common';
import { fixabilityLabels } from 'constants/reportConstants';
import { defaultEmailBody, defaultEmailBodyWithNoCVEsFound } from '../forms/emailTemplateFormUtils';
import { ReportParametersFormValues } from '../forms/useReportFormValues';
import { getCVEsDiscoveredSinceText, imageTypeLabelMap } from '../utils';

export type EmailTemplatePreviewProps = {
    emailSubject: string;
    emailBody: string;
    defaultEmailSubject: string;
    reportParameters: ReportParametersFormValues;
};

function EmailTemplatePreview({
    emailSubject,
    emailBody,
    defaultEmailSubject,
    reportParameters,
}: EmailTemplatePreviewProps) {
    const [selectedPreviewText, setSelectedPreviewText] = useState<string>('CVEs found');

    return (
        <Flex
            className="pf-u-py-lg"
            spaceItems={{ default: 'spaceItemsMd' }}
            direction={{ default: 'column' }}
        >
            <FlexItem>
                <TextContent>
                    <Text component={TextVariants.small}>
                        This preview displays modifications to the email subject and body only. Data
                        shown in the report parameters are sample data meant solely for
                        illustration. For any actual data, please check the email attachment in the
                        real report. Please not that an attachment of the report data will not be
                        provided if no CVEs are found.
                    </Text>
                </TextContent>
            </FlexItem>
            <FlexItem>
                <ToggleGroup aria-label="Preview with or without CVEs found">
                    <ToggleGroupItem
                        text="CVEs found"
                        isSelected={selectedPreviewText === 'CVEs found'}
                        onChange={() => setSelectedPreviewText('CVEs found')}
                    />
                    <ToggleGroupItem
                        text="CVEs not found"
                        isSelected={selectedPreviewText === 'CVEs not found'}
                        onChange={() => setSelectedPreviewText('CVEs not found')}
                    />
                </ToggleGroup>
            </FlexItem>
            <FlexItem>
                <Card isFlat>
                    <CardTitle>{emailSubject || defaultEmailSubject}</CardTitle>
                    <CardBody>
                        {emailBody ||
                            (selectedPreviewText === 'CVEs found'
                                ? defaultEmailBody
                                : defaultEmailBodyWithNoCVEsFound)}
                    </CardBody>
                    <CardFooter>
                        {/* NOTE: When using this in plain HTML, replace the style object with a style string like this: style="padding: 0 0 10px 0;" */}
                        <div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    Config name:
                                </span>
                                <span>{reportParameters.reportName}</span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    Number of CVEs found:
                                </span>
                                <span>
                                    {selectedPreviewText === 'CVEs found'
                                        ? '# in Deployed images; # in Watched images'
                                        : '0 in Deployed images; 0 in Watched images'}
                                </span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    CVE severity:
                                </span>
                                <span>
                                    {reportParameters.cveSeverities
                                        .map((cveSeverity) => {
                                            return vulnerabilitySeverityLabels[cveSeverity];
                                        })
                                        .join(', ')}
                                </span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    CVE status:
                                </span>
                                <span>
                                    {reportParameters.cveStatus
                                        .map((status) => {
                                            return fixabilityLabels[status];
                                        })
                                        .join(', ')}
                                </span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    Report scope:
                                </span>
                                <span>{reportParameters.reportScope?.name || '-'}</span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    Image type:
                                </span>
                                <span>
                                    {reportParameters.imageType
                                        .map((type) => {
                                            return imageTypeLabelMap[type];
                                        })
                                        .join(', ')}
                                </span>
                            </div>
                            <div style={{ padding: '0 0 10px 0' }}>
                                <span style={{ fontWeight: 'bold', marginRight: '10px' }}>
                                    CVEs discovered since:
                                </span>
                                <span>{getCVEsDiscoveredSinceText(reportParameters)}</span>
                            </div>
                        </div>
                    </CardFooter>
                </Card>
            </FlexItem>
        </Flex>
    );
}

export default EmailTemplatePreview;
