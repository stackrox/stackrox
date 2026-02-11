import type { ReactElement } from 'react';
import {
    Badge,
    Card,
    CardFooter,
    CardHeader,
    CardTitle,
    Flex,
    GalleryItem,
    Truncate,
} from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom-v5-compat';

import TechPreviewLabel from 'Components/PatternFly/PreviewLabel/TechPreviewLabel';

type IntegrationTileProps = {
    categories?: string;
    image: string;
    label: string;
    linkTo: string;
    numIntegrations: number;
    isTechPreview?: boolean;
};

function IntegrationTile({
    categories,
    image,
    label,
    linkTo,
    numIntegrations,
    isTechPreview = false,
}: IntegrationTileProps): ReactElement {
    const navigate = useNavigate();

    return (
        <GalleryItem>
            <Card isClickable isCompact data-testid="integration-tile">
                <CardHeader
                    actions={{
                        actions: <>{numIntegrations > 0 && <Badge>{numIntegrations}</Badge>}</>,
                        // This is needed to reverse the order of the actions, aligning the badge to the right
                        // and preventing a gap caused by the navigation `selectableAction` (which is invisible, but takes space).
                        className: 'pf-v6-u-flex-direction-row-reverse',
                    }}
                    selectableActions={{
                        onClickAction: () => navigate(linkTo),
                        selectableActionAriaLabel: `View ${label} integrations`,
                    }}
                    className="pf-v6-u-mb-lg"
                >
                    <>
                        <img src={image} alt="" style={{ height: '100px' }} />
                    </>
                </CardHeader>
                <CardTitle className="pf-v6-u-color-100" style={{ whiteSpace: 'nowrap' }}>
                    <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                        <Truncate position="middle" content={label} />
                        {isTechPreview && <TechPreviewLabel />}
                    </Flex>
                </CardTitle>
                {categories && <CardFooter className="pf-v6-u-color-200">{categories}</CardFooter>}
            </Card>
        </GalleryItem>
    );
}

export default IntegrationTile;
