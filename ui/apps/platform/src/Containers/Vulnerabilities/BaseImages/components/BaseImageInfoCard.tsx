import React from 'react';
import { Button, Card, CardBody, Flex, FlexItem, Label, Text } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { vulnerabilitiesBasePath } from 'routePaths';

export type BaseImageInfo = {
    name: string;
    isTracked: boolean;
    baseImageId?: string;
};

type BaseImageInfoCardProps = {
    baseImage: BaseImageInfo;
    onTrackBaseImage?: (baseImageName: string) => void;
};

/**
 * Card component that displays base image information on the Image Details page
 * Shows tracking status and provides actions to view or track the base image
 */
function BaseImageInfoCard({ baseImage, onTrackBaseImage }: BaseImageInfoCardProps) {
    const { name, isTracked, baseImageId } = baseImage;

    const handleTrackClick = () => {
        if (onTrackBaseImage) {
            onTrackBaseImage(name);
        }
    };

    return (
        <Card>
            <CardBody>
                <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem>
                        <Text component="small" className="pf-v5-u-color-200">
                            Base Image
                        </Text>
                        <Text component="h3" className="pf-v5-u-font-size-xl pf-v5-u-mt-sm">
                            {name}
                        </Text>
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Flex
                            direction={{ default: 'row' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            <FlexItem>
                                {isTracked ? (
                                    <Label color="green">Tracked</Label>
                                ) : (
                                    <Label color="grey">Not Tracked</Label>
                                )}
                            </FlexItem>
                            <FlexItem>
                                {isTracked && baseImageId ? (
                                    <Button
                                        variant="secondary"
                                        component={(props) => (
                                            <Link
                                                {...props}
                                                to={`${vulnerabilitiesBasePath}/base-images/${baseImageId}`}
                                            />
                                        )}
                                    >
                                        View base image
                                    </Button>
                                ) : (
                                    <Button variant="primary" onClick={handleTrackClick}>
                                        Track this base image
                                    </Button>
                                )}
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                </Flex>
            </CardBody>
        </Card>
    );
}

export default BaseImageInfoCard;
