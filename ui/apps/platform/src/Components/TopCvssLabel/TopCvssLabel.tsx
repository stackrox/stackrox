import React, { ReactNode } from 'react';

export type TopCvssLabelProps = {
    cvss: number;
    version: string;
    expanded?: boolean;
};

function TopCvssLabel({ cvss, version, expanded }: TopCvssLabelProps): ReactNode {
    // Early return here might be redundant with similar if statement preceding function call.
    if (typeof cvss !== 'number') {
        return 'N/A';
    }

    const cvss1 = cvss.toFixed(1);

    if (expanded) {
        // Vertical key value pairs for entity overview page.
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

    // Horizontal score and version for entities table cell.
    // Replace mr-2 with mr-1 and space for PDF Export.
    return (
        <span>
            <span data-testid="label-chip" className="mr-1">
                {cvss1}
            </span>
            <span className="text-xs"> {version}</span>
        </span>
    );
}

export default TopCvssLabel;
