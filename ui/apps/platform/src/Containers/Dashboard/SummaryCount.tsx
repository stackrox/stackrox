import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Button, Stack } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

export type SummaryCountProps = {
    count: number;
    href: string;
    noun: string;
};

function SummaryCount({ count, href, noun }: SummaryCountProps): ReactElement {
    return (
        <Button variant="link" component={LinkShim} href={href}>
            <Stack className="pf-u-px-xs pf-u-px-sm-on-xl pf-u-align-items-center">
                <span className="pf-u-font-size-lg-on-md pf-u-font-size-sm pf-u-font-weight-bold">
                    {count}
                </span>
                <span className="pf-u-font-size-md-on-md pf-u-font-size-xs">
                    {pluralize(noun, count)}
                </span>
            </Stack>
        </Button>
    );
}

export default SummaryCount;
