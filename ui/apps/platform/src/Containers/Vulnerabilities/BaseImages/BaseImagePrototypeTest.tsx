import React, { useState } from 'react';
import { PageSection, Stack, StackItem, Title, Divider } from '@patternfly/react-core';
import PageTitle from 'Components/PageTitle';
import BaseImageInfoCard from './components/BaseImageInfoCard';
import LayerTypeBadge from './components/LayerTypeBadge';
import {
    extractBaseImageInfo,
    isBaseImageTracked,
    getBaseImageId,
    isFromBaseImageLayer,
    estimateLastBaseImageLayerIndex,
} from './utils/imageBaseLayerUtils';

/**
 * Test page for Phase 3 components
 * Navigate to: /vulnerabilities/base-images/test
 */
function BaseImagePrototypeTest() {
    const [trackedImages] = useState(['ubuntu:22.04', 'alpine:3.18']);

    // Test data
    const testImageNames = [
        'myapp/frontend:latest', // Based on ubuntu
        'myapp/backend:v1.2.3', // Based on node
        'nginx:1.24',
        'redis:7.0',
    ];

    const testImageName = 'ubuntu:22.04-myapp';

    // Extract base image info
    const baseImageInfo = extractBaseImageInfo(testImageName);
    const isTracked = baseImageInfo ? isBaseImageTracked(baseImageInfo.name, trackedImages) : false;
    const baseImageId = baseImageInfo ? getBaseImageId(baseImageInfo.name) : undefined;

    // Update base image info with tracking status
    const fullBaseImageInfo = baseImageInfo
        ? {
              ...baseImageInfo,
              isTracked,
              baseImageId,
          }
        : null;

    // Test layer detection
    const lastBaseLayerIndex = estimateLastBaseImageLayerIndex(testImageName);
    const testComponentLayers = [
        { name: 'libc', layerIndex: 2, isBase: true },
        { name: 'openssl', layerIndex: 4, isBase: true },
        { name: 'myapp-binary', layerIndex: 6, isBase: false },
        { name: 'config-files', layerIndex: 7, isBase: false },
    ];

    const handleTrackImage = (imageName: string) => {
        // eslint-disable-next-line no-alert
        alert(`Would track base image: ${imageName}`);
    };

    return (
        <>
            <PageTitle title="Base Image Prototype Test" />
            <PageSection variant="light">
                <Title headingLevel="h1">Phase 3 Component Test Page</Title>
            </PageSection>

            <PageSection>
                <Stack hasGutter>
                    <StackItem>
                        <Title headingLevel="h2">1. BaseImageInfoCard Component</Title>
                        <p>
                            Testing with image: <code>{testImageName}</code>
                        </p>
                        {fullBaseImageInfo && (
                            <BaseImageInfoCard
                                baseImage={fullBaseImageInfo}
                                onTrackBaseImage={handleTrackImage}
                            />
                        )}
                        {!fullBaseImageInfo && <p>No base image detected in this image name</p>}
                    </StackItem>

                    <Divider />

                    <StackItem>
                        <Title headingLevel="h2">2. LayerTypeBadge Component</Title>
                        <Stack hasGutter>
                            <StackItem>
                                <p>Base Image Badge:</p>
                                <LayerTypeBadge layerType="base" />
                            </StackItem>
                            <StackItem>
                                <p>Application Badge:</p>
                                <LayerTypeBadge layerType="application" />
                            </StackItem>
                        </Stack>
                    </StackItem>

                    <Divider />

                    <StackItem>
                        <Title headingLevel="h2">3. Base Image Detection Utils</Title>
                        <Stack hasGutter>
                            {testImageNames.map((imageName) => {
                                const info = extractBaseImageInfo(imageName);
                                return (
                                    <StackItem key={imageName}>
                                        <code>{imageName}</code>
                                        {info ? (
                                            <>
                                                {' → '}
                                                <strong>{info.name}</strong>
                                            </>
                                        ) : (
                                            ' → No base image detected'
                                        )}
                                    </StackItem>
                                );
                            })}
                        </Stack>
                    </StackItem>

                    <Divider />

                    <StackItem>
                        <Title headingLevel="h2">4. Layer Type Detection</Title>
                        <p>
                            Last base layer index for <code>{testImageName}</code>:{' '}
                            <strong>{lastBaseLayerIndex}</strong>
                        </p>
                        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
                            <thead>
                                <tr style={{ borderBottom: '1px solid #ccc' }}>
                                    <th style={{ textAlign: 'left', padding: '8px' }}>Component</th>
                                    <th style={{ textAlign: 'left', padding: '8px' }}>
                                        Layer Index
                                    </th>
                                    <th style={{ textAlign: 'left', padding: '8px' }}>
                                        Is Base Layer?
                                    </th>
                                    <th style={{ textAlign: 'left', padding: '8px' }}>Badge</th>
                                </tr>
                            </thead>
                            <tbody>
                                {testComponentLayers.map((component) => {
                                    const isBase = isFromBaseImageLayer(
                                        component.layerIndex,
                                        lastBaseLayerIndex
                                    );
                                    return (
                                        <tr
                                            key={component.name}
                                            style={{ borderBottom: '1px solid #eee' }}
                                        >
                                            <td style={{ padding: '8px' }}>{component.name}</td>
                                            <td style={{ padding: '8px' }}>
                                                {component.layerIndex}
                                            </td>
                                            <td style={{ padding: '8px' }}>
                                                {isBase ? '✅ Yes' : '❌ No'}
                                            </td>
                                            <td style={{ padding: '8px' }}>
                                                <LayerTypeBadge
                                                    layerType={isBase ? 'base' : 'application'}
                                                />
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </StackItem>

                    <Divider />

                    <StackItem>
                        <Title headingLevel="h2">5. Tracking Status</Title>
                        <p>Currently tracked base images:</p>
                        <ul>
                            {trackedImages.map((img) => (
                                <li key={img}>
                                    <code>{img}</code>
                                </li>
                            ))}
                        </ul>
                    </StackItem>
                </Stack>
            </PageSection>
        </>
    );
}

export default BaseImagePrototypeTest;
