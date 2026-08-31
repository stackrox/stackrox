import { Flex, FlexItem } from '@patternfly/react-core';
import type { ThProps } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';
import { getVersionedDocs } from 'utils/versioning';

export const infoForEpssProbability: ThProps['info'] = {
    // ariaLabel for OutlinedQuestionCircleIcon
    ariaLabel: 'Information about EPSS probability',
    // PopoverBodyContent replaces 5 issues with 1 from axe DevTools:
    // https://dequeuniversity.com/rules/axe/4.10/aria-dialog-name
    // Popover element does not have aria=labelledby attribute
    // rendered if there is a popoverProps.headerContent property.
    popover: (
        <PopoverBodyContent
            headerContent="EPSS probability"
            bodyContent={
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>Likelihood of exploitability</FlexItem>
                    <FlexItem>
                        For more information, see{' '}
                        <ExternalLink>
                            <a
                                href="https://www.first.org/epss/"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Exploit Prediction Scoring System (EPSS)
                            </a>
                        </ExternalLink>
                    </FlexItem>
                </Flex>
            }
        />
    ),
};

// The CVE origin popover links to versioned RHACS documentation, so it must be
// built with the current version rather than defined as a static object.
export function getInfoForCveOrigin(
    version: string,
    resourceType: 'image' | 'deployment'
): ThProps['info'] {
    return {
        ariaLabel: 'Information about CVE origin',
        popover: (
            <PopoverBodyContent
                headerContent="CVE origin"
                bodyContent={
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            Where the CVE assessment comes from for components in this{' '}
                            {resourceType}. Values are supported OS vendors, OSV.dev, Other, or
                            Multiple when components differ.
                        </FlexItem>
                        <FlexItem>
                            For more information, see{' '}
                            <ExternalLink>
                                <a
                                    href={getVersionedDocs(
                                        version,
                                        'operating/examine-images-for-vulnerabilities#supported-operating-systems_examine-images-for-vulnerabilities'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    Supported operating systems (RHACS)
                                </a>
                            </ExternalLink>
                        </FlexItem>
                    </Flex>
                }
            />
        ),
    };
}
