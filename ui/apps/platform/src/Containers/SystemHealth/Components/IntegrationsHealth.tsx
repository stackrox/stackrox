import React, { ReactElement } from 'react';
import { getDateTime } from 'utils/dateUtils';

import { styleHealthy, styleUnhealthy } from 'Containers/Clusters/cluster.helpers';

import { CountableText, getCountableText, style0 } from '../utils/health';
import { IntegrationMergedItem } from '../utils/integrations';

type Props = {
    healthyText: CountableText;
    integrationsMerged: IntegrationMergedItem[];
};

const IntegrationsHealth = ({ healthyText, integrationsMerged }: Props): ReactElement => {
    let nHealthy = 0;
    const integrationsFiltered: IntegrationMergedItem[] = [];

    integrationsMerged.forEach((integrationMergedItem) => {
        switch (integrationMergedItem.status) {
            case 'HEALTHY':
                nHealthy += 1;
                break;

            case 'UNHEALTHY':
                integrationsFiltered.push(integrationMergedItem);
                break;

            default:
        }
    });

    // The item border matches the widget header border.
    if (integrationsFiltered.length !== 0) {
        const { Icon, fgColor } = styleUnhealthy;
        return (
            <ul className="leading-normal pt-1 w-full">
                {integrationsFiltered.map(({ id, name, type, errorMessage, lastTimestamp }) => (
                    <li className="border-b border-base-300 px-2 py-1" key={id}>
                        <div className="flex w-full">
                            <div className={`flex-shrink-0 ${fgColor}`}>
                                <Icon className="h-4 w-4" />
                            </div>
                            <div className="ml-2 flex-grow">
                                <div className="flex justify-between">
                                    <span className="font-700">{name}</span>
                                    <span className="italic text-base-500">{type}</span>
                                </div>
                                {errorMessage && (
                                    <div>
                                        <span>Error message:</span> <span>{errorMessage}</span>
                                    </div>
                                )}
                                {lastTimestamp && (
                                    <div>
                                        <span>Last contact:</span>{' '}
                                        <span>{getDateTime(lastTimestamp)}</span>
                                    </div>
                                )}
                            </div>
                        </div>
                    </li>
                ))}
            </ul>
        );
    }

    const { Icon, fgColor } = nHealthy === 0 ? style0 : styleHealthy;
    const text = `${nHealthy} ${getCountableText(healthyText, nHealthy)}`;

    return (
        <div className={`flex flex-col h-full justify-center w-full ${fgColor}`}>
            <div className="flex justify-center mb-2">
                <Icon className="h-6 w-6" />
            </div>
            <div className="leading-normal px-2 text-center">{text}</div>
        </div>
    );
};

export default IntegrationsHealth;
