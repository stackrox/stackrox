import React, { ReactElement } from 'react';
import { DescriptionList, Divider } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

function ContainerSecrets({ secrets }): ReactElement {
    return (
        <>
            {secrets.map(({ name, path }, idx) => (
                <div key={path}>
                    <DescriptionList isHorizontal>
                        <DescriptionListItem term="Name" desc={name} />
                        <DescriptionListItem term="Container path" desc={path} />
                    </DescriptionList>

                    {idx !== secrets.length - 1 && (
                        <Divider component="div" className="pf-u-py-md" />
                    )}
                </div>
            ))}
        </>
    );
}

export default ContainerSecrets;
