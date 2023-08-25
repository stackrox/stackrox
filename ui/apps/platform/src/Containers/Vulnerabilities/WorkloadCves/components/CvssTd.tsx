import React from 'react';

export type CvssTdProps = {
    cvss: number;
    scoreVersion?: string;
};

function CvssTd({ cvss, scoreVersion }: CvssTdProps) {
    return (
        <>
            {cvss.toFixed(1)} {scoreVersion ? `(${scoreVersion})` : null}
        </>
    );
}

export default CvssTd;
