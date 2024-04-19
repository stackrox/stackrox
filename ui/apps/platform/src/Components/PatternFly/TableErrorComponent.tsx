import React from 'react';
import { Bullseye } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';

type TableErrorComponentProps = {
    error: Error;
    message: string;
};

// TODO: Consider renaming this component to CenteredError or something similar, since it can be used in non-table cases
function TableErrorComponent({ error, message }: TableErrorComponentProps) {
    return (
        <Bullseye>
            <EmptyStateTemplate
                headingLevel="h2"
                title={getAxiosErrorMessage(error)}
                icon={ExclamationCircleIcon}
                iconClassName="pf-v5-u-danger-color-100"
            >
                {message}
            </EmptyStateTemplate>
        </Bullseye>
    );
}

export default TableErrorComponent;
