import React from 'react';
import { ThProps } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';

export const infoForEpssProbability: ThProps['info'] = {
    ariaLabel: 'Information about EPSS probability',
    popover: <>Information to be determined</>,
    popoverProps: {
        headerContent: 'EPSS probability',
        footerContent: (
            <>
                For more information, see{' '}
                <ExternalLink>
                    <a href="https://www.first.org/epss/" target="_blank" rel="noopener noreferrer">
                        Exploit Prediction Scoring System (EPSS)
                    </a>
                </ExternalLink>
            </>
        ),
    },
};
