import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { getDateTime } from 'utils/dateUtils';

import { styleHealthy, styleUnhealthy } from 'Containers/Clusters/cluster.helpers';

import { IntegrationMergedItem } from '../utils/integrations';

type Props = {
    integrationsMerged: IntegrationMergedItem[];
};

const IntegrationsHealth = ({ integrationsMerged }: Props): ReactElement => {
    const nHealthy = 0;
    const integrationsFiltered: IntegrationMergedItem[] = [];

    integrationsMerged.forEach((integrationMergedItem) => {
        integrationsFiltered.push(integrationMergedItem);
    });

    // The item border matches the widget header border.
    if (integrationsFiltered.length !== 0) {
        const { Icon, fgColor } = styleUnhealthy;
        return (
            <ul className="leading-normal pt-1 w-full">
                {integrationsFiltered.map(({ id, name, label, lastTimestamp }) => (
                    <li className="border-b border-base-300 px-2 py-1" key={id}>
                        <div className="flex w-full">
                            <div className={`flex-shrink-0 ${fgColor}`}>
                                <Icon className="h-4 w-4" />
                            </div>
                            <div className="ml-2 flex-grow">
                                <div className="font-700" data-testid="integration-name">
                                    {name}
                                </div>
                                {label && label !== name && (
                                    <div className="text-base-500" data-testid="integration-label">
                                        {label}
                                    </div>
                                )}
                                {lastTimestamp && (
                                    <div>
                                        <span>Last contact:</span>{' '}
                                        <span data-testid="last-contact">
                                            {getDateTime(lastTimestamp)}
                                        </span>
                                    </div>
                                )}
                            </div>
                        </div>
                    </li>
                ))}
            </ul>
        );
    }

    const { Icon, fgColor } = styleHealthy;
    const nIntegrations = integrationsMerged.length;
    let text = 'No configured integrations';
    if (nIntegrations !== 0) {
        if (nHealthy === nIntegrations) {
            text = `${nHealthy} healthy ${pluralize('integration', nHealthy)}`;
        } else {
            text = `${nHealthy} / ${nIntegrations} healthy integrations`; // cannot both be singular
        }
    }

    return (
        <div className={`flex flex-col h-full justify-center w-full ${fgColor}`}>
            <div className="flex justify-center mb-2">
                <Icon className="h-6 w-6" />
            </div>
            <div className="leading-normal px-2 text-center" data-testid="healthy-text">
                {text}
            </div>
        </div>
    );
};

export default IntegrationsHealth;
