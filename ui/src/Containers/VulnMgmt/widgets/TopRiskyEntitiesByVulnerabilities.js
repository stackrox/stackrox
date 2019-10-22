import React from 'react';
import Widget from 'Components/Widget';

const TopRiskyEntitiesByVulnerabilities = () => {
    const header = 'Top Risky Deployments By CVE Count & CVSS Score';
    return (
        <Widget className="h-full pdf-page" header={header}>
            <div />
        </Widget>
    );
};

export default TopRiskyEntitiesByVulnerabilities;
