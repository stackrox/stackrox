import React from 'react';

export type CveSelectionsProps = {
    cves: string[];
};

function CveSelections({ cves }: CveSelectionsProps) {
    return (
        <div>
            {cves.map((cve) => (
                <p key={cve}>{cve}</p>
            ))}
        </div>
    );
}

export default CveSelections;
