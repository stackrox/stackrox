import React from 'react';
import { healthStatusStyles } from '../../cluster.helpers';
import { healthStatusLabels } from '../../../../messages/common';

const trClassName = 'align-bottom leading-normal'; // align-bottom in case heading text wraps
const thClassName = 'font-600 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-right';
const tdErrorsClassName = 'font-600 pb-0 pl-0 pr-1 pt-2 text-left'; // pt for gap above errors

type ScannerStatusTotalsProps = {
    localScannerHealthInfo: {
        totalDesiredAnalyzerPods: number;
        totalReadyAnalyzerPods: number;
        totalDesiredDbPods: number;
        totalReadyDbPods: number;
        statusErrors: string[];
    };
};

const notAvailable = 'n/a';

const resolveDbHealthStatus = (desiredPods: number, readyPods: number) => {
    if (!desiredPods) {
        return notAvailable;
    }
    return healthStatusLabels[readyPods ? 'HEALTHY' : 'UNHEALTHY'];
};

const ScannerStatusTotals = ({ localScannerHealthInfo }: ScannerStatusTotalsProps) => {
    const {
        totalDesiredAnalyzerPods,
        totalReadyAnalyzerPods,
        totalDesiredDbPods,
        totalReadyDbPods,
        statusErrors,
    } = localScannerHealthInfo;

    return (
        <table data-testid="localScannerHealthInfo">
            <tbody>
                <tr className={trClassName} key="totalReadyPods">
                    <th className={thClassName} scope="row">
                        Scanner pods ready:
                    </th>
                    <td className={tdClassName} data-testid="totalReadyPods">
                        <span>
                            {totalReadyAnalyzerPods === null
                                ? notAvailable
                                : totalReadyAnalyzerPods}
                        </span>
                    </td>
                </tr>
                <tr className={trClassName} key="totalDesiredPods">
                    <th className={thClassName} scope="row">
                        Scanner pods expected:
                    </th>
                    <td className={tdClassName} data-testid="totalDesiredPods">
                        {totalDesiredAnalyzerPods === null
                            ? notAvailable
                            : totalDesiredAnalyzerPods}
                    </td>
                </tr>
                <tr className={trClassName} key="dbAvailable">
                    <th className={thClassName} scope="row">
                        Database:
                    </th>
                    <td className={tdClassName} data-testid="dbAvailable">
                        {resolveDbHealthStatus(totalDesiredDbPods, totalReadyDbPods)}
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
};

export default ScannerStatusTotals;
