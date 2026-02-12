import type { ReactElement } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { PageSection, Tab, TabTitleText, Tabs, Title } from '@patternfly/react-core';
import pluralize from 'pluralize';

import type { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { entityPathSegment, getEntityPath } from './accessControlPaths';

export type AccessControlHeadingProps = {
    /** The AccessControl Entity managed on this page, used to highlight the current navigation item. */
    entityType?: AccessControlEntityType;
};

/**
 * Render title h1 and tab navigation at top of page.
 */
function AccessControlHeading({ entityType }: AccessControlHeadingProps): ReactElement {
    const navigate = useNavigate();
    const entityTypes = Object.keys(entityPathSegment) as AccessControlEntityType[];
    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Access Control</Title>
            </PageSection>
            <PageSection type="tabs">
                <Tabs
                    activeKey={entityType}
                    onSelect={(_event, tabKey) => {
                        navigate(getEntityPath(tabKey as AccessControlEntityType));
                    }}
                    usePageInsets
                    mountOnEnter
                    unmountOnExit
                >
                    {entityTypes.map((itemType) => (
                        <Tab
                            key={itemType}
                            eventKey={itemType}
                            title={
                                <TabTitleText>
                                    {pluralize(accessControlLabels[itemType])}
                                </TabTitleText>
                            }
                            tabContentId={itemType}
                        />
                    ))}
                </Tabs>
            </PageSection>
        </>
    );
}

export default AccessControlHeading;
