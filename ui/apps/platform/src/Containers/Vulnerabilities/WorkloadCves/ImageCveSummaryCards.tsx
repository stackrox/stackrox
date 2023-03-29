import React from 'react';
import { gql } from '@apollo/client';
import { Flex } from '@patternfly/react-core';
import { VulnerabilitySeverity } from 'types/cve.proto';
import AffectedImages from './SummaryCards/AffectedImages';
import TopCvssScoreBreakdown from './SummaryCards/TopCvssScoreBreakdown';
import BySeveritySummaryCard from './SummaryCards/BySeveritySummaryCard';

export type ImageCveSummaryCount = {
    totalImageCount: number;
};

export type ImageCveSeveritySummary = {
    affectedImageCountBySeverity: {
        critical: number;
        important: number;
        moderate: number;
        low: number;
    };
    affectedImageCount: number;
    topCVSS: number;
};

export const imageCveSeveritySummaryFragment = gql`
    fragment ImageCVESeveritySummary on ImageCVECore {
        # TODO These need to accept query parameters to be filtered correctly
        affectedImageCountBySeverity {
            critical
            important
            moderate
            low
        }
        affectedImageCount
        topCVSS
        # TODO vector
    }
`;

export const imageCveSummaryCountFragment = gql`
    fragment ImageCVESummaryCounts on Query {
        totalImageCount: imageCount
    }
`;

export type ImageCveSummaryCardsProps = {
    summaryCounts: ImageCveSummaryCount;
    severitySummary: ImageCveSeveritySummary;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

function ImageCveSummaryCards({
    summaryCounts,
    severitySummary,
    hiddenSeverities,
}: ImageCveSummaryCardsProps) {
    const { critical, important, moderate, low } = severitySummary.affectedImageCountBySeverity;
    const { affectedImageCount, topCVSS } = severitySummary;
    const { totalImageCount } = summaryCounts;
    return (
        <Flex
            direction={{ default: 'column', lg: 'row' }}
            alignItems={{ lg: 'alignItemsStretch' }}
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
        >
            <AffectedImages
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                // TODO Should affectedImagesCount be the value derived from
                //      `imageCount(query: $query)` or from `affectedImageCount`?
                affectedImageCount={affectedImageCount}
                totalImagesCount={totalImageCount}
            />
            <BySeveritySummaryCard
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                title="Images by severity"
                severityCounts={{
                    CRITICAL_VULNERABILITY_SEVERITY: critical,
                    IMPORTANT_VULNERABILITY_SEVERITY: important,
                    MODERATE_VULNERABILITY_SEVERITY: moderate,
                    LOW_VULNERABILITY_SEVERITY: low,
                }}
                hiddenSeverities={hiddenSeverities}
            />
            <TopCvssScoreBreakdown
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                cvssScore={topCVSS}
                vector="TODO - Not implemented"
            />
        </Flex>
    );
}

export default ImageCveSummaryCards;
