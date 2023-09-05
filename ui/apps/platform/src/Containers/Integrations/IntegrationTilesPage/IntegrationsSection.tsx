import React, { ReactElement } from 'react';
import { PageSection, Title, Gallery, GalleryItem } from '@patternfly/react-core';

type IntegrationsSectionProps = {
    headerName: string;
    children: ReactElement[];
    id: string;
};

const IntegrationsSection = ({
    headerName,
    children,
    id,
}: IntegrationsSectionProps): ReactElement => {
    const galleryItems = React.Children.map(children, (child) => {
        return <GalleryItem>{child}</GalleryItem>;
    });
    return (
        <PageSection variant="light" id={id} className="pf-u-mb-xl">
            <div className="pf-u-mb-md">
                <Title headingLevel="h2">{headerName}</Title>
            </div>
            <Gallery hasGutter>{galleryItems}</Gallery>
        </PageSection>
    );
};

export default IntegrationsSection;
