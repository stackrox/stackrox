import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';
import React, { ReactElement } from 'react';

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
    const isIncludeEpssProbabilityEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const hasIncludeEpssProbability =
        isIncludeEpssProbabilityEnabled && formValues.reportParameters.includeEpssProbability;
    const isIncludeNvdCvssEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const hasIncludeNvdCvss = isIncludeNvdCvssEnabled && formValues.reportParameters.includeNvdCvss;

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
                    {(hasIncludeNvdCvss || hasIncludeEpssProbability) && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Optional columns</DescriptionListTerm>
                            {hasIncludeNvdCvss && (
                                <DescriptionListDescription>
                                    Include NVD CVSS
                                </DescriptionListDescription>
                            )}
                            {hasIncludeEpssProbability && (
                                <DescriptionListDescription>
                                    Include EPSS probability
                                </DescriptionListDescription>
                            )}
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default ReportParametersDetails;
