import React from 'react';

export type CvssFormattedProps = {
    cvss: number;
    scoreVersion?: string;
};

function CvssFormatted({ cvss, scoreVersion }: CvssFormattedProps) {
    return (
        <>
            {cvss.toFixed(1)} {scoreVersion ? `(${scoreVersion})` : null}
        </>
    );
}

export default CvssFormatted;
