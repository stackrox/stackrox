import React from 'react';
import { Label } from '@patternfly/react-core';
import { severityColorMapPF } from 'constants/severityColors';
import { severityLabels } from 'messages/common';
import { getSeverityByCvss } from 'utils/vulnerabilityUtils';

export type CVSSScoreLabelProps = {
    cvss: string;
    scoreVersion?: string;
};

function CVSSScoreLabel({ cvss, scoreVersion }: CVSSScoreLabelProps) {
    const severity = getSeverityByCvss(cvss);
    const severityLabel = severityLabels[severity];
    const cvssNum = Number(cvss).toFixed(1);

    return (
        <>
            <Label color={severityColorMapPF[severityLabel]}>{cvssNum}</Label>
            {scoreVersion && <span className="pf-u-ml-sm">({scoreVersion})</span>}
        </>
    );
}

export default CVSSScoreLabel;
