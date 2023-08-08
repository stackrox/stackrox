import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { portExposureLabels } from 'messages/common';
import { PortConfig } from 'types/deployment.proto';

type PortDescriptionListProps = {
    port: PortConfig;
};

function PortDescriptionList({ port }: PortDescriptionListProps): ReactElement {
    const { name, containerPort, protocol, exposure, exposureInfos } = port;

    /* eslint-disable react/no-array-index-key */
    return (
        <DescriptionList isCompact isHorizontal>
            {name && <DescriptionListItem term="name" desc={name} />}
            <DescriptionListItem term="containerPort" desc={containerPort} />
            <DescriptionListItem term="protocol" desc={protocol} />
            <DescriptionListItem
                term="exposure"
                desc={portExposureLabels[exposure] || portExposureLabels.UNSET}
            />
            {exposureInfos.map((exposureInfo, i) => {
                const {
                    level,
                    serviceName,
                    serviceId,
                    serviceClusterIp,
                    servicePort,
                    nodePort,
                    externalIps,
                    externalHostnames,
                } = exposureInfo;
                return (
                    <DescriptionListItem
                        key={i}
                        term={`exposureInfo[${i}]`}
                        desc={
                            <DescriptionList isCompact isHorizontal>
                                {level && <DescriptionListItem term="level" desc={level} />}
                                {serviceName && (
                                    <DescriptionListItem term="serviceName" desc={serviceName} />
                                )}
                                {serviceId && (
                                    <DescriptionListItem term="serviceId" desc={serviceId} />
                                )}
                                {serviceClusterIp && (
                                    <DescriptionListItem
                                        term="serviceClusterIp"
                                        desc={serviceClusterIp}
                                    />
                                )}
                                {typeof servicePort === 'number' && (
                                    <DescriptionListItem term="servicePort" desc={servicePort} />
                                )}
                                {typeof nodePort === 'number' && (
                                    <DescriptionListItem term="nodePort" desc={nodePort} />
                                )}
                                {Array.isArray(externalIps) && externalIps.length !== 0 && (
                                    <DescriptionListItem
                                        term="externalIps"
                                        desc={externalIps.join(', ')}
                                    />
                                )}
                                {Array.isArray(externalHostnames) &&
                                    externalHostnames.length !== 0 && (
                                        <DescriptionListItem
                                            term="externalHostnames"
                                            desc={externalHostnames.join(', ')}
                                        />
                                    )}
                            </DescriptionList>
                        }
                    />
                );
            })}
        </DescriptionList>
    );
    /* eslint-enable react/no-array-index-key */
}

export default PortDescriptionList;
