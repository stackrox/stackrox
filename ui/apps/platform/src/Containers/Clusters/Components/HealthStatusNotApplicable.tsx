import type { ReactElement } from 'react';

type HealthStatusNotApplicableProps = { testId: string; isList?: boolean };

function HealthStatusNotApplicable({
    testId,
    isList = false,
}: HealthStatusNotApplicableProps): ReactElement {
    return (
        <div className={isList ? 'pf-v6-u-display-inline' : ''} data-testid={testId}>
            <span className="pf-v6-u-text-nowrap">Not applicable</span>
        </div>
    );
}

export default HealthStatusNotApplicable;
