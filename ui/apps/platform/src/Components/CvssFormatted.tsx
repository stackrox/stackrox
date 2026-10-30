import type { ReactElement } from 'react';

export type CvssFormattedProps = {
    cvss: number | string | null | undefined;
    scoreVersion?: string;
};

function CvssFormatted({ cvss, scoreVersion }: CvssFormattedProps): ReactElement {
    if (typeof cvss !== 'number' || !Number.isFinite(cvss) || scoreVersion === 'UNKNOWN_VERSION') {
        return <>Not available</>;
    }

    const cvssFormatted = cvss.toFixed(1);

    return <>{scoreVersion ? `${cvssFormatted} (${scoreVersion})` : cvssFormatted}</>;
}

export default CvssFormatted;
