import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';
import { ExternalLinkSquareAltIcon } from '@patternfly/react-icons';

type MitreAttackLinkProps = {
    href: string;
    id: string;
};

function MitreAttackLink({ href, id }: MitreAttackLinkProps): ReactElement {
    /* eslint-disable jsx-a11y/control-has-associated-label, jsx-a11y/anchor-has-content */
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
    /* eslint-enable jsx-a11y/control-has-associated-label, jsx-a11y/anchor-has-content */
}

export default MitreAttackLink;
