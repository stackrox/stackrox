import React, { ReactElement } from 'react';

import { healthStatusStyles } from '../../cluster.helpers';

const trClassName = 'align-bottom leading-normal'; // align-bottom in case heading text wraps
const thClassName = 'font-700 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-right';
const tdErrorsClassName = 'pb-0 pl-0 pr-1 pt-2 text-left'; // pt for gap above errors

type CollectorStatusTotalsProps = {
    collectorHealthInfo: {
        totalReadyPods: number;
        totalDesiredPods: number;
        totalRegisteredNodes: number;
        version: string;
        statusErrors: string[];
    };
};

function CollectorStatusTotals({ collectorHealthInfo }: CollectorStatusTotalsProps): ReactElement {
    const notAvailable = 'n/a';
    const { totalReadyPods, totalDesiredPods, totalRegisteredNodes, version, statusErrors } =
        collectorHealthInfo;
    return (
        <table data-testid="collectorHealthInfo">
            <tbody>
                <tr className={trClassName} key="version">
                    <th className={thClassName} scope="row">
                        Collector version:
                    </th>
                    <td className={`${tdClassName} break-all`} data-testid="version">
                        {version || notAvailable}
                    </td>
                </tr>
                <tr className={trClassName} key="totalReadyPods">
                    <th className={thClassName} scope="row">
                        Collector pods ready:
                    </th>
                    <td className={tdClassName} data-testid="totalReadyPods">
                        <span>{totalReadyPods == null ? notAvailable : totalReadyPods}</span>
                    </td>
                </tr>
                <tr className={trClassName} key="totalDesiredPods">
                    <th className={thClassName} scope="row">
                        Collector pods expected:
                    </th>
                    <td className={tdClassName} data-testid="totalDesiredPods">
                        {totalDesiredPods == null ? notAvailable : totalDesiredPods}
                    </td>
                </tr>
                <tr className={trClassName} key="totalRegisteredNodes">
                    <th className={thClassName} scope="row">
                        Registered nodes in cluster:
                    </th>
                    <td className={tdClassName} data-testid="totalRegisteredNodes">
                        {totalRegisteredNodes == null ? notAvailable : totalRegisteredNodes}
                    </td>
                </tr>
                {statusErrors && statusErrors.length > 0 && (
                    <tr className={trClassName} key="statusErrors">
                        <td className={tdErrorsClassName} colSpan={2} data-testid="statusErrors">
                            <ul>
                                {statusErrors.map((err) => (
                                    <li key={err}>
                                        <span
                                            className={`${healthStatusStyles.UNHEALTHY.fgColor} break-all`}
                                        >
                                            {err}
                                        </span>
                                    </li>
                                ))}
                            </ul>
                        </td>
                    </tr>
                )}
            </tbody>
        </table>
    );
}

export default CollectorStatusTotals;
