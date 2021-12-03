import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { NotifierIntegration } from 'types/notifier.proto';

import { getNotifierTypeLabel } from '../policies.utils';

type NotifierProps = {
    notifierId: string;
    notifiers: NotifierIntegration[];
};

function Notifier({ notifierId, notifiers }: NotifierProps): ReactElement {
    const notifier = notifiers.find(({ id }) => id === notifierId);
    const typeLabel = getNotifierTypeLabel(notifier?.type ?? '');
    return (
        <DescriptionList isCompact isHorizontal>
            {notifier?.name ? (
                <DescriptionListItem term="Name" desc={notifier.name} />
            ) : (
                <DescriptionListItem term="Id" desc={notifierId} />
            )}
            {typeLabel && <DescriptionListItem term="Type" desc={typeLabel} />}
        </DescriptionList>
    );
}

export default Notifier;
