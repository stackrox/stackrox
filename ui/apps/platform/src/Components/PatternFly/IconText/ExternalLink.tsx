import React, { ReactElement, ReactNode } from 'react';
import { Flex } from '@patternfly/react-core';
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
            <ExternalLinkAltIcon color="var(--pf-global--link--Color)" />
        </Flex>
    );
}

export default ExternalLink;
