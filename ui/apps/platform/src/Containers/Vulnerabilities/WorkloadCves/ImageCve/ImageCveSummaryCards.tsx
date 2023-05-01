import React from 'react';
import { gql } from '@apollo/client';
import { Flex } from '@patternfly/react-core';
import { VulnerabilitySeverity } from 'types/cve.proto';
import AffectedImages from '../SummaryCards/AffectedImages';
import TopCvssScoreBreakdown from '../SummaryCards/TopCvssScoreBreakdown';
import BySeveritySummaryCard from '../SummaryCards/BySeveritySummaryCard';

export type ImageCveSummaryCount = {
    totalImageCount: number;
};

export type ImageCveSeveritySummary = {
    affectedImageCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    affectedImageCount: number;
    topCVSS: number;
};

export const imageCveSeveritySummaryFragment = gql`
    fragment ImageCVESeveritySummary on ImageCVECore {
        affectedImageCountBySeverity {
            critical {
                total
            }
            important {
                total
            }
            moderate {
                total
            }
            low {
                total
            }
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
    severitySummary: ImageCveSeveritySummary | null;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

function ImageCveSummaryCards({
    summaryCounts,
    severitySummary,
    hiddenSeverities,
}: ImageCveSummaryCardsProps) {
    const { critical, important, moderate, low } =
        severitySummary?.affectedImageCountBySeverity ?? {};
    const { affectedImageCount, topCVSS } = severitySummary ?? {};
    const { totalImageCount } = summaryCounts;
    return (
        <Flex
            direction={{ default: 'column', lg: 'row' }}
            alignItems={{ lg: 'alignItemsStretch' }}
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
        >
            <AffectedImages
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                affectedImageCount={affectedImageCount ?? 0}
                totalImagesCount={totalImageCount}
            />
            <BySeveritySummaryCard
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                title="Images by severity"
                severityCounts={{
                    CRITICAL_VULNERABILITY_SEVERITY: critical?.total ?? 0,
                    IMPORTANT_VULNERABILITY_SEVERITY: important?.total ?? 0,
                    MODERATE_VULNERABILITY_SEVERITY: moderate?.total ?? 0,
                    LOW_VULNERABILITY_SEVERITY: low?.total ?? 0,
                }}
                hiddenSeverities={hiddenSeverities}
            />
            <TopCvssScoreBreakdown
                className="pf-u-flex-grow-1 pf-u-flex-basis-0"
                cvssScore={topCVSS ?? 0}
                vector="TODO - Not implemented"
            />
        </Flex>
    );
}

export default ImageCveSummaryCards;
