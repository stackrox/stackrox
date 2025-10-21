import React from 'react';
import type { ReactElement } from 'react';
import { ExclamationCircleIcon, ExclamationTriangleIcon } from '@patternfly/react-icons';
import { Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import { Icon } from '@patternfly/react-core';

type TheadClustersHealthProps = {
    dataLabelHealthy?: string;
    dataLabelUnhealthy?: string;
    dataLabelDegraded?: string;
};

export function TheadClustersHealth({
    dataLabelHealthy,
    dataLabelUnhealthy,
    dataLabelDegraded,
}: TheadClustersHealthProps): ReactElement {
    return (
        <Thead>
            <Tr>
                <Th width={35}>
                    <span className="pf-v5-screen-reader">Clusters</span>
                </Th>
                <Th width={10} className="pf-v5-u-text-align-right">
                    {dataLabelHealthy || 'Healthy'}
                </Th>
                <Th width={10} className="pf-v5-u-text-align-right">
                    {dataLabelUnhealthy || 'Unhealthy'}
                </Th>
                <Th width={10} className="pf-v5-u-text-align-right">
                    {dataLabelDegraded || 'Degraded'}
                </Th>
                <Th width={10} className="pf-v5-u-text-align-right">
                    Unavailable
                </Th>
                <Th width={10} className="pf-v5-u-text-align-right">
                    Uninitialized
                </Th>
                <Th width={15} className="pf-v5-u-text-align-right">
                    Total
                </Th>
            </Tr>
        </Thead>
    );
}

// Component props have inconsistent name but we will postpone improvements to minimize code churm.
/* eslint-disable generic/react-props-name */
type TdStatusWithDataLabelProps = {
    count: number;
    dataLabel?: string;
};

export function TdHealthy({ count, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel={dataLabel || 'Healthy'}>
            {count}
        </Td>
    );
}

export function TdUnhealthy({ count, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel={dataLabel || 'Unhealthy'}>
            {count !== 0 ? (
                <IconText
                    icon={
                        <Icon>
                            <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                        </Icon>
                    }
                    text={String(count)}
                />
            ) : (
                <>{count}</>
            )}
        </Td>
    );
}

export function TdDegraded({ count, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel={dataLabel || 'Degraded'}>
            {count !== 0 ? (
                <IconText
                    icon={
                        <Icon>
                            <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />
                        </Icon>
                    }
                    text={String(count)}
                />
            ) : (
                <>{count}</>
            )}
        </Td>
    );
}

type TdStatusWithoutDataLabelProps = {
    count: number;
};

export function TdUnavailable({ count }: TdStatusWithoutDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel="Unavailable">
            {count}
        </Td>
    );
}

export function TdUninitialized({ count }: TdStatusWithoutDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel="Uninitialized">
            {count}
        </Td>
    );
}

export function TdTotal({ count }: TdStatusWithoutDataLabelProps): ReactElement {
    return (
        <Td className="pf-v5-u-text-align-right" dataLabel="Total">
            {count}
        </Td>
    );
}
/* eslint-enable generic/react-props-name */
