import React from 'react';
import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import { fixabilityLabels } from 'constants/reportConstants';
import {
    getCVEsDiscoveredSinceText,
    imageTypeLabelMap,
} from 'Containers/Vulnerabilities/VulnerablityReporting/utils';

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import useFeatureFlags from 'hooks/useFeatureFlags';

export type ReportParametersDetailsProps = {
    headingLevel: 'h2' | 'h3';
    formValues: ReportFormValues;
};

function ReportParametersDetails({
    headingLevel,
    formValues,
}: ReportParametersDetailsProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const optionalColumnsDescriptions: ReactElement[] = [];
    if (isFeatureFlagEnabled('ROX_SCANNER_V4') && formValues.reportParameters.includeNvdCvss) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeNvdCvss">NVDCVSS</DescriptionListDescription>
        );
    }
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        formValues.reportParameters.includeEpssProbability
    ) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeEpssProbability">
                EPSS Probability Percentage
            </DescriptionListDescription>
        );
    }
    if (isFeatureFlagEnabled('ROX_SCANNER_V4') && formValues.reportParameters.includeAdvisory) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeAdvisory">
                Advisory Name and Advisory Link
            </DescriptionListDescription>
        );
    }
    /*
    // Ross CISA KEV includeKnownExploit?
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_WHATEVER') &&
        formValues.reportParameters.includeKnownExploit
    ) {
        optionalColumnsDescriptions.push(
            <DescriptionListDescription key="includeKnownExploit">
                Known exploit
            </DescriptionListDescription>
        );
    }
    */

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

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Title headingLevel={headingLevel}>Report parameters</Title>
            </FlexItem>
            <FlexItem flex={{ default: 'flexNone' }}>
                <DescriptionList
                    isFillColumns
                    columnModifier={{
                        default: '3Col',
                        md: '3Col',
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
                        <DescriptionListTerm>Report description</DescriptionListTerm>
                        <DescriptionListDescription>
                            {formValues.reportParameters.reportDescription || 'None'}
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
                        <DescriptionListTerm>Collection included</DescriptionListTerm>
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
                            {getCVEsDiscoveredSinceText(formValues.reportParameters)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Non-optional columns</DescriptionListTerm>
                        <DescriptionListDescription>Cluster</DescriptionListDescription>
                        <DescriptionListDescription>Namespace</DescriptionListDescription>
                        <DescriptionListDescription>Deployment</DescriptionListDescription>
                        <DescriptionListDescription>Image</DescriptionListDescription>
                        <DescriptionListDescription>Component</DescriptionListDescription>
                        <DescriptionListDescription>CVE</DescriptionListDescription>
                        <DescriptionListDescription>Fixable</DescriptionListDescription>
                        <DescriptionListDescription>CVE Fixed In</DescriptionListDescription>
                        <DescriptionListDescription>Severity</DescriptionListDescription>
                        <DescriptionListDescription>CVSS</DescriptionListDescription>
                        <DescriptionListDescription>Discovered At</DescriptionListDescription>
                        <DescriptionListDescription>Reference</DescriptionListDescription>
                    </DescriptionListGroup>
                    {optionalColumnsDescriptions.length !== 0 && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Optional columns</DescriptionListTerm>
                            {optionalColumnsDescriptions}
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default ReportParametersDetails;
