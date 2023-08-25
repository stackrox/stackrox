import React, { ReactElement } from 'react';

type HealthStatusNotApplicableProps = { testId: string; isList?: boolean };

function HealthStatusNotApplicable({
    testId,
    isList = false,
}: HealthStatusNotApplicableProps): ReactElement {
    return (
        <div className={`${isList ? 'inline' : ''} leading-normal`} data-testid={testId}>
            <span className="pf-u-text-nowrap">Not applicable</span>
        </div>
    );
}

export default HealthStatusNotApplicable;
