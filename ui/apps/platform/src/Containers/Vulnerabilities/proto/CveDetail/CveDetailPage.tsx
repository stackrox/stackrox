import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
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

import { vulnerabilitiesPrototypeCvePath } from 'routePaths';

import { useCveDetail } from './useCveDetail';
import LayoutToggle, { useLayoutMode } from './LayoutToggle';
import FlowLayout from './FlowLayout';
import TabLayout from './TabLayout';
import CollapsibleLayout from './CollapsibleLayout';

/**
 * CVE detail page wrapper. Fetches advisory data for the CVE specified in the
 * URL parameter and renders the selected layout variant.
 */
function CveDetailPage() {
    const { cveName } = useParams<{ cveName: string }>();
    const { data, loading, error } = useCveDetail(cveName ?? '');
    const [layoutMode, setLayoutMode] = useLayoutMode();

    const advisories = data?.protoCVEDetail ?? [];
    const topAdvisory = advisories[0];

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Breadcrumb>
                    <BreadcrumbItem>
                        <Link to={vulnerabilitiesPrototypeCvePath}>CVE Prototype</Link>
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
                            {topAdvisory && (
                                <FlexItem>
                                    <Label color="blue">CVSS {topAdvisory.cvss.toFixed(1)}</Label>
                                </FlexItem>
                            )}
                        </Flex>
                        {topAdvisory?.description && <p>{topAdvisory.description}</p>}
                    </StackItem>

                    <StackItem>
                        <Flex>
                            <FlexItem>
                                <LayoutToggle mode={layoutMode} onSelect={setLayoutMode} />
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
                    <FlowLayout advisories={advisories} />
                )}
                {!loading && !error && layoutMode === 'tabs' && (
                    <TabLayout advisories={advisories} />
                )}
                {!loading && !error && layoutMode === 'collapsible' && (
                    <CollapsibleLayout advisories={advisories} />
                )}
            </PageSection>
        </>
    );
}

export default CveDetailPage;
