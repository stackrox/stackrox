import type { ReactElement } from 'react';

export type CvssFormattedProps = {
    cvss: number;
    scoreVersion?: string;
};

function CvssFormatted({ cvss, scoreVersion }: CvssFormattedProps): ReactElement {
    if (scoreVersion === 'UNKNOWN_VERSION') {
        // For NVD CVSS.
        return <>Not available</>;
    }

    const cvssFormatted = cvss.toFixed(1);

    return <>{scoreVersion ? `${cvssFormatted} (${scoreVersion})` : cvssFormatted}</>;
}

export default CvssFormatted;
