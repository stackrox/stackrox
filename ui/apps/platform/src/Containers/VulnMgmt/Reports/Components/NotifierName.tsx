import React, { ReactElement } from 'react';
import { Spinner } from '@patternfly/react-core';

import useFetchNotifiers from 'hooks/useFetchNotifiers';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type NotifierNameProps = {
    notifierId: string;
};

function NotifierName({ notifierId }: NotifierNameProps): ReactElement {
    const notifiersResult = useFetchNotifiers();

    const fullNotifier = notifiersResult.notifiers.find((notifier) => notifier.id === notifierId);

    if (notifiersResult.isLoading) {
        return <Spinner isSVG size="md" />;
    }

    if (notifiersResult.error) {
        return (
            <span>Error getting notifier info. {getAxiosErrorMessage(notifiersResult.error)}</span>
        );
    }

    return <span>{fullNotifier?.name}</span> || <em>No notifier specified</em>;
}

export default NotifierName;
