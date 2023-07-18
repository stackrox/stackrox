import React, { ReactElement } from 'react';

import { healthStatusStyles } from '../../cluster.helpers';

const trClassName = 'align-bottom leading-normal'; // align-bottom in case heading text wraps
const thClassName = 'font-700 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-right';
const tdErrorsClassName = 'pb-0 pl-0 pr-1 pt-2 text-left'; // pt for gap above errors

type AdmissionControlStatusTotalsProps = {
    admissionControlHealthInfo: {
        totalReadyPods: number;
        totalDesiredPods: number;
        statusErrors: string[];
    };
};

function AdmissionControlStatusTotals({
    admissionControlHealthInfo,
}: AdmissionControlStatusTotalsProps): ReactElement {
    const notAvailable = 'n/a';
    const { totalReadyPods, totalDesiredPods, statusErrors } = admissionControlHealthInfo;
    return (
        <table data-testid="admissionControlHealthInfo">
            <tbody>
                <tr className={trClassName} key="totalReadyPods">
                    <th className={thClassName} scope="row">
                        Admission Control pods ready:
                    </th>
                    <td className={tdClassName} data-testid="totalReadyPods">
                        <span>{totalReadyPods == null ? notAvailable : totalReadyPods}</span>
                    </td>
                </tr>
                <tr className={trClassName} key="totalDesiredPods">
                    <th className={thClassName} scope="row">
                        Admission Control pods expected:
                    </th>
                    <td className={tdClassName} data-testid="totalDesiredPods">
                        {totalDesiredPods == null ? notAvailable : totalDesiredPods}
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

export default AdmissionControlStatusTotals;
