import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Content,
    Divider,
    Flex,
    FlexItem,
    Label,
    PageSection,
    Spinner,
    Stack,
    StackItem,
    Title,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import { vulnerabilitiesPrototypePath } from 'routePaths';

import { useCveDetail } from './useCveDetail';
import LayoutToggle, { useLayoutMode } from './LayoutToggle';
import FlowLayout from './FlowLayout';
import TabLayout from './TabLayout';
import CollapsibleLayout from './CollapsibleLayout';

const severityNames: Record<number, string> = {
    0: 'Unknown',
    1: 'Low',
    2: 'Moderate',
    3: 'Important',
    4: 'Critical',
};

/**
 * CVE detail page wrapper. Fetches CVE data from the REST API and renders
 * the selected layout variant.
 */
function CveDetailPage() {
    const { cveName } = useParams<{ cveName: string }>();
    const { data, loading, error } = useCveDetail(cveName ?? '');
    const [layoutMode, setLayoutMode] = useLayoutMode();

    const advisories = data?.advisories ?? [];
    const components = data?.components ?? [];
    const images = data?.images ?? [];

    const cvssDisplay = data?.cvss ? data.cvss.toFixed(1) : '-';
    const severityDisplay = severityNames[data?.severity ?? 0] ?? 'Unknown';

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Breadcrumb>
                    <BreadcrumbItem>
                        <Link to={vulnerabilitiesPrototypePath}>
                            Vulnerability Management V5
                        </Link>
                    </BreadcrumbItem>
                    <BreadcrumbItem>
                        <Link to={vulnerabilitiesPrototypePath + '/cves'}>
                            CVEs
                        </Link>
                    </BreadcrumbItem>
                    <BreadcrumbItem isActive>{cveName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>

            <PageSection hasBodyWrapper={false}>
                <Stack hasGutter>
                    <StackItem>
                        <Flex>
                            <FlexItem>
                                <Title headingLevel="h1">{cveName}</Title>
                            </FlexItem>
                            {data && (
                                <>
                                    <FlexItem>
                                        <Label color="blue">
                                            CVSS {cvssDisplay}
                                        </Label>
                                    </FlexItem>
                                    <FlexItem>
                                        <Label color="grey">
                                            {severityDisplay}
                                        </Label>
                                    </FlexItem>
                                </>
                            )}
                        </Flex>
                    </StackItem>

                    {data?.description && (
                        <StackItem>
                            <Content component="p">{data.description}</Content>
                        </StackItem>
                    )}

                    <StackItem>
                        <Flex>
                            <FlexItem>
                                <LayoutToggle
                                    mode={layoutMode}
                                    onSelect={setLayoutMode}
                                />
                            </FlexItem>
                        </Flex>
                    </StackItem>
                </Stack>
            </PageSection>

            <Divider />

            <PageSection hasBodyWrapper={false}>
                {loading && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}

                {error && <p>Error loading CVE detail: {error.message}</p>}

                {!loading && !error && layoutMode === 'flow' && (
                    <FlowLayout
                        advisories={advisories}
                        components={components}
                        images={images}
                    />
                )}
                {!loading && !error && layoutMode === 'tabs' && (
                    <TabLayout
                        advisories={advisories}
                        components={components}
                        images={images}
                    />
                )}
                {!loading && !error && layoutMode === 'collapsible' && (
                    <CollapsibleLayout
                        advisories={advisories}
                        components={components}
                        images={images}
                    />
                )}
            </PageSection>
        </>
    );
}

export default CveDetailPage;
