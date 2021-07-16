import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';
import {
    BanIcon,
    CheckIcon,
    ExclamationTriangleIcon,
    LongArrowAltDownIcon,
    LongArrowAltUpIcon,
} from '@patternfly/react-icons';

import { EffectiveAccessScopeState } from 'services/AccessScopesService';

const notAllowedColor = 'var(--pf-global--danger-color--100)';
const allowedColor = 'var(--pf-global--success-color--100)';
const unknownColor = 'var(--pf-global--warning-color--100)';

/*
 * Tooltip has key prop to replace the previous tooltip if status changes.
 */

const notAllowedIcon = <BanIcon color={notAllowedColor} />;
const notAllowedCluster = (
    <Tooltip
        key="notAllowedCluster"
        content={
            <div>
                Not allowed: cluster
                <br />
                nor any of its namespaces
            </div>
        }
        isContentLeftAligned
    >
        {notAllowedIcon}
    </Tooltip>
);
const notAllowedNamespace = (
    <Tooltip key="notAllowedNamespace" content="Not allowed: namespace">
        {notAllowedIcon}
    </Tooltip>
);

const allowedIcon = <CheckIcon color={allowedColor} />;
const allowedCluster = (
    <Tooltip
        key="allowedCluster"
        content={
            <div>
                Allowed: cluster
                <br />
                and therefore all of its namespaces
            </div>
        }
        isContentLeftAligned
    >
        <span>
            {allowedIcon}
            <LongArrowAltDownIcon color={allowedColor} style={{ transform: 'rotate(-45deg)' }} />
        </span>
    </Tooltip>
);
const allowedNamespace = (
    <Tooltip key="allowedNamespace" content="Allowed: namespace">
        {allowedIcon}
    </Tooltip>
);

const partialCluster = (
    <Tooltip
        key="partialCluster"
        content={
            <div>
                Conditionally allowed: cluster
                <br />
                because at least one of its namespaces
            </div>
        }
        isContentLeftAligned
    >
        <span>
            {allowedIcon}
            <LongArrowAltUpIcon color={allowedColor} style={{ transform: 'rotate(-45deg)' }} />
        </span>
    </Tooltip>
);

const unknownState = (
    <Tooltip key="unknownState" content="Unknown">
        <ExclamationTriangleIcon color={unknownColor} />
    </Tooltip>
);

export type EffectiveAccessScopeStateProps = {
    state: EffectiveAccessScopeState;
    isCluster: boolean;
};

function EffectiveAccessScopeStateIcon({
    state,
    isCluster,
}: EffectiveAccessScopeStateProps): ReactElement {
    switch (state) {
        case 'EXCLUDED':
            return isCluster ? notAllowedCluster : notAllowedNamespace;

        case 'INCLUDED':
            return isCluster ? allowedCluster : allowedNamespace;

        case 'PARTIAL':
            return partialCluster;

        default:
            return unknownState;
    }
}

export default EffectiveAccessScopeStateIcon;
