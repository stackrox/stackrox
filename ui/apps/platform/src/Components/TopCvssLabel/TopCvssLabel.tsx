import React, { ReactNode } from 'react';

function TopCvssLabel({ cvss, version, expanded }: TopCvssLabelProps): ReactNode {
    if (!cvss && cvss !== 0) {
        return 'N/A';
    }

    const cvss1 = cvss.toFixed(1);

    if (!expanded) {
        return (
            <span>
                <span>{cvss1}</span> <span className="text-xs">({version})</span>
            </span>
        );
    }

    return (
        <div className="flex flex-col">
            <div>
                <span className="font-700 mr-2">Top CVSS:</span>
                <span>{cvss1}</span>
            </div>
            <div>
                <span className="font-700 mr-2">CVSS Version:</span>
                <span>{version}</span>
            </div>
        </div>
    );
}

type TopCvssLabelProps = {
    cvss: number;
    version: string;
    expanded?: boolean;
};

export default TopCvssLabel;
