import { Flex, Icon, Tooltip, Truncate } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import type { SourceType } from 'types/image.proto';

export type ComponentLocationProps = {
    location: string;
    source: SourceType;
};

function ComponentLocation({ location, source }: ComponentLocationProps) {
    return (
        <Flex spaceItems={{ default: 'spaceItemsXs' }} alignItems={{ default: 'alignItemsCenter' }}>
            {location ? (
                <Truncate content={location} position="middle" />
            ) : (
                <>
                    <span>N/A</span>
                    {source === 'OS' && (
                        <Tooltip content="Location is unavailable for operating system packages">
                            <Icon>
                                <InfoCircleIcon
                                    color="var(--pf-t--temp--dev--tbd)" /* CODEMODS: original v5 color was --pf-v5-global--info-color--100 */
                                />
                            </Icon>
                        </Tooltip>
                    )}
                </>
            )}
        </Flex>
    );
}

export default ComponentLocation;
