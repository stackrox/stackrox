import React from 'react';
import { ExpandableSection, List, ListItem } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { CollectionParseError } from './converter';

export type UnsupportedCollectionStateProps = {
    errors: CollectionParseError['errors'];
    className?: string;
};

function UnsupportedCollectionState({ errors, className = '' }: UnsupportedCollectionStateProps) {
    return (
        <div className={className}>
            <EmptyStateTemplate
                title="This collection cannot be displayed"
                headingLevel="h2"
                icon={InfoCircleIcon}
            >
                <p className="pf-u-pb-lg">
                    This collection is valid but cannot be displayed nor edited through the user
                    interface
                </p>
                <ExpandableSection toggleText="More info">
                    <List className="pf-u-text-align-left">
                        {errors.map((err) => (
                            <ListItem key={err}>{err}</ListItem>
                        ))}
                    </List>
                </ExpandableSection>
            </EmptyStateTemplate>
        </div>
    );
}

export default UnsupportedCollectionState;
