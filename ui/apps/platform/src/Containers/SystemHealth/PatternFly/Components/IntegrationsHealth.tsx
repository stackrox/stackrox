import React, { ReactElement, ComponentClass } from 'react';
import { getDateTime } from 'utils/dateUtils';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import { Bullseye, Divider, Flex, FlexItem } from '@patternfly/react-core';

import { IntegrationMergedItem } from '../utils/integrations';

type Props = {
    unhealthyIntegrations: IntegrationMergedItem[];
    fgColor: string;
    Icon: ComponentClass<SVGIconProps>;
};

const IntegrationsHealth = ({ unhealthyIntegrations, fgColor, Icon }: Props): ReactElement => {
    if (unhealthyIntegrations.length !== 0) {
        return (
            <ul className="">
                {unhealthyIntegrations.map(({ id, name, label, lastTimestamp }, i) => (
                    <li key={id}>
                        {i > 0 && <Divider component="div" />}
                        <Flex
                            alignItems={{ default: 'alignItemsFlexStart' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                            className="pf-u-py-sm"
                        >
                            <FlexItem className={fgColor}>
                                <Icon size="md" />
                            </FlexItem>
                            <FlexItem>
                                <div
                                    className="pf-u-font-weight-bold"
                                    data-testid="integration-name"
                                >
                                    {name}
                                </div>
                                {label && label !== name && (
                                    <div
                                        className="pf-u-font-weight-light"
                                        data-testid="integration-label"
                                    >
                                        <em>{label}</em>
                                    </div>
                                )}
                                {lastTimestamp && (
                                    <div className="pf-u-text-wrap">
                                        <span>Last contact:</span>{' '}
                                        <span data-testid="last-contact">
                                            {getDateTime(lastTimestamp)}
                                        </span>
                                    </div>
                                )}
                            </FlexItem>
                        </Flex>
                    </li>
                ))}
            </ul>
        );
    }

    return (
        <Bullseye className={`pf-u-py-md ${fgColor}`}>
            <Icon size="lg" />
        </Bullseye>
    );
};

export default IntegrationsHealth;
