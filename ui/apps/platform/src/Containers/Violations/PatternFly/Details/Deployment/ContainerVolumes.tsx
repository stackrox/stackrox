import React, { ReactElement } from 'react';
import { Divider, DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

function ContainerVolumes({ volumes }): ReactElement {
    return (
        <>
            {volumes.map((volume, idx) => (
                <>
                    <DescriptionList isHorizontal>
                        <DescriptionListItem term="Name" desc={volume.name} />
                        {volume.source && (
                            <DescriptionListItem term="Source" desc={volume.source} />
                        )}
                        <DescriptionListItem term="Destination" desc={volume.destination} />
                        {volume.readOnly && <DescriptionListItem term="Read Only" desc="true" />}
                        <DescriptionListItem term="Type" desc={volume.type} />
                        <DescriptionListItem
                            term="Mount Propagation"
                            desc={volume.mountPropagation}
                        />
                    </DescriptionList>

                    {idx !== volumes.length - 1 && (
                        <Divider component="div" className="pf-u-py-md" />
                    )}
                </>
            ))}
        </>
    );
}

export default ContainerVolumes;
