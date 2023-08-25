import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';
import { ExternalLinkSquareAltIcon } from '@patternfly/react-icons';

type MitreAttackLinkProps = {
    href: string;
    id: string;
};

function MitreAttackLink({ href, id }: MitreAttackLinkProps): ReactElement {
    return (
        <Button
            variant="link"
            isInline
            component="a"
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            icon={<ExternalLinkSquareAltIcon />}
            iconPosition="right"
        >
            {id}
        </Button>
    );
}

export default MitreAttackLink;
