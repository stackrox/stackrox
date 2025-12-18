import type { ReactElement } from 'react';

import { vulnerabilitySeverityColorMap } from 'constants/severityColors';
import { getPercentage } from 'utils/mathUtils';
import type { VulnerabilitySeverity } from 'types/cve.proto';

export type SeverityStackedPillProps = {
    vulnCounter: VulnCounter;
};

export type VulnCounter = {
    all: {
        fixable: number;
        total: number;
    };
    critical: {
        fixable: number;
        total: number;
    };
    important: {
        fixable: number;
        total: number;
    };
    moderate: {
        fixable: number;
        total: number;
    };
    low: {
        fixable: number;
        total: number;
    };
};

const vulnKeyMap = {
    low: 'LOW_VULNERABILITY_SEVERITY',
    moderate: 'MODERATE_VULNERABILITY_SEVERITY',
    important: 'IMPORTANT_VULNERABILITY_SEVERITY',
    critical: 'CRITICAL_VULNERABILITY_SEVERITY',
} as const;

type VulnKey = keyof typeof vulnKeyMap;

function SeverityStackedPill({ vulnCounter }: SeverityStackedPillProps): ReactElement {
    const { total } = vulnCounter.all;

    return (
        <div
            className="flex rounded-full w-full min-w-10 max-w-24 h-3 bg-base-300"
            style={{ boxShadow: 'inset 0 0px 8px 0 hsla(0, 0%, 0%, .10) !important' }}
        >
            {(Object.entries(vulnKeyMap) as [VulnKey, VulnerabilitySeverity][])
                .filter(([dataKey]) => vulnCounter[dataKey].total !== 0)
                .map(([dataKey, colorKey]) => (
                    <div
                        key={dataKey}
                        className="border-r border-base-100"
                        style={{
                            backgroundColor: vulnerabilitySeverityColorMap[colorKey],
                            width: `${getPercentage(vulnCounter[dataKey].total, total)}%`,
                        }}
                    />
                ))}
        </div>
    );
}

export default SeverityStackedPill;
