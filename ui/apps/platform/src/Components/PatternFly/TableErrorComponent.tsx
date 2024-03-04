import React from 'react';
import { Bullseye } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

type TableErrorComponentProps = {
    error: Error;
    message: string;
};

function TableErrorComponent({ error, message }: TableErrorComponentProps) {
    return (
        <Bullseye>
            <EmptyStateTemplate
                headingLevel="h2"
                title={getAxiosErrorMessage(error)}
                icon={ExclamationCircleIcon}
                iconClassName="pf-u-danger-color-100"
            >
                {message}
            </EmptyStateTemplate>
        </Bullseye>
    );
}

export default TableErrorComponent;
