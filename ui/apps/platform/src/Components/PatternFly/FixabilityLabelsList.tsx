import React, { ReactElement } from 'react';

import { fixabilityLabels } from 'constants/reportConstants';
import { getMappedFixability } from 'Containers/VulnMgmt/Reports/VulnMgmtReport.utils';
import { Fixability } from 'types/report.proto';

export type FixabilityLabelsListProps = {
    fixability: Fixability;
};

function FixabilityLabelsList({ fixability: value }: FixabilityLabelsListProps): ReactElement {
    const mappedFixabilityValues = getMappedFixability(value);

    const fixabilityStrings = mappedFixabilityValues.map((fixValue) => fixabilityLabels[fixValue]);

    return (
        <span>
            {fixabilityStrings.length > 0 ? (
                fixabilityStrings.join(', ')
            ) : (
                <em>No fixability specified</em>
            )}
        </span>
    );
}

export default FixabilityLabelsList;
