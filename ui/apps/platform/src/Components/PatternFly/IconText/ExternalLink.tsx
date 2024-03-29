import React, { ReactElement, ReactNode } from 'react';
import { Flex, Icon } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

export type ExternalLinkProps = {
    children: ReactNode;
};

/*
 * Pure presentation component for links that open in a new tab:
 * docs page
 * product page (for example, collection)
 * vulnerability description
 */
function ExternalLink({ children }: ExternalLinkProps): ReactElement {
    return (
        <Flex
            alignItems={{ default: 'alignItemsCenter' }}
            display={{ default: 'inlineFlex' }}
            spaceItems={{ default: 'spaceItemsSm' }}
        >
            {children}
            <Icon>
                <ExternalLinkAltIcon color="var(--pf-v5-global--link--Color)" />
            </Icon>
        </Flex>
    );
}

export default ExternalLink;
