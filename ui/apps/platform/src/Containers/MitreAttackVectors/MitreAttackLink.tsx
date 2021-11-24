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
            component={(props) => (
                <a {...props} href={href} target="_blank" rel="noopener noreferrer" />
            )}
            icon={<ExternalLinkSquareAltIcon />}
            iconPosition="right"
            isInline
        >
            {id}
        </Button>
    );
    /* eslint-enable jsx-a11y/control-has-associated-label, jsx-a11y/anchor-has-content */
}

export default MitreAttackLink;
