import React from 'react';
import Widget from 'Components/Widget';

const TopRiskyEntitiesByVulnerabilities = () => {
    const header = 'Top Risky Deployments By CVE Count & CVSS Score';
    return (
        <Widget className="sx-4 sy-2 pdf-page" header={header}>
            <div />
        </Widget>
    );
};

export default TopRiskyEntitiesByVulnerabilities;
