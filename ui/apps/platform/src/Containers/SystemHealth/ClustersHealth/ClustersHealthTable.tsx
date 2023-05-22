import React, { ReactElement } from 'react';
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
} from '@patternfly/react-icons';
import { Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';

import { ClusterStatusCounts } from './ClustersHealth.utils';

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
                <Th width={35} />
                <Th width={10} className="pf-u-text-align-right">
                    {dataLabelHealthy || 'Healthy'}
                </Th>
                <Th width={10} className="pf-u-text-align-right">
                    {dataLabelUnhealthy || 'Unhealthy'}
                </Th>
                <Th width={10} className="pf-u-text-align-right">
                    {dataLabelDegraded || 'Degraded'}
                </Th>
                <Th width={10} className="pf-u-text-align-right">
                    Unavailable
                </Th>
                <Th width={10} className="pf-u-text-align-right">
                    Uninitialized
                </Th>
                <Th width={15} className="pf-u-text-align-right">
                    Total
                </Th>
            </Tr>
        </Thead>
    );
}

type TdStatusWithDataLabelProps = {
    counts: ClusterStatusCounts;
    dataLabel?: string;
};

export function TdHealthy({ counts, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    const count = counts.HEALTHY;

    return (
        <Td className="pf-u-text-align-right" dataLabel={dataLabel || 'Healthy'}>
            {count !== 0 && counts.UNHEALTHY === 0 && counts.DEGRADED === 0 ? (
                <IconText
                    icon={<CheckCircleIcon color="var(--pf-global--success-color--100)" />}
                    text={String(count)}
                />
            ) : (
                <>{count}</>
            )}
        </Td>
    );
}

export function TdUnhealthy({ counts, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    const count = counts.UNHEALTHY;

    return (
        <Td className="pf-u-text-align-right" dataLabel={dataLabel || 'Unhealthy'}>
            {count !== 0 ? (
                <IconText
                    icon={<ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />}
                    text={String(count)}
                />
            ) : (
                <>{counts.UNHEALTHY}</>
            )}
        </Td>
    );
}

export function TdDegraded({ counts, dataLabel }: TdStatusWithDataLabelProps): ReactElement {
    const count = counts.DEGRADED;

    return (
        <Td className="pf-u-text-align-right" dataLabel={dataLabel || 'Degraded'}>
            {count !== 0 ? (
                <IconText
                    icon={<ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" />}
                    text={String(count)}
                />
            ) : (
                <>{count}</>
            )}
        </Td>
    );
}

type TdStatusWithoutDataLabelProps = {
    counts: ClusterStatusCounts;
};

export function TdUnavailable({ counts }: TdStatusWithoutDataLabelProps): ReactElement {
    return (
        <Td className="pf-u-text-align-right" dataLabel="Unavailable">
            {counts.UNAVAILABLE}{' '}
        </Td>
    );
}

export function TdUninitialized({ counts }: TdStatusWithoutDataLabelProps): ReactElement {
    return (
        <Td className="pf-u-text-align-right" dataLabel="Uninitialized">
            {counts.UNINITIALIZED}{' '}
        </Td>
    );
}

type TdTotalProps = {
    clusters: unknown[];
};

export function TdTotal({ clusters }: TdTotalProps): ReactElement {
    return (
        <Td className="pf-u-text-align-right" dataLabel="Total">
            {clusters.length}
        </Td>
    );
}
