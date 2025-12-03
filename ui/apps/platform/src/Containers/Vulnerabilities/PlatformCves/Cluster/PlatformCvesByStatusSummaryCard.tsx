import { gql } from '@apollo/client';
import { Card, CardBody, CardTitle, Content, Flex, pluralize } from '@patternfly/react-core';
import { MinusIcon, WrenchIcon } from '@patternfly/react-icons';
import type { FixableStatus } from '../../types';

const disabledColor = 'var(--pf-t--global--text--color--disabled)';

const statusDisplays = [
    {
        status: 'Fixable',
        Icon: WrenchIcon,
        text: ({ fixable }: PlatformCVECountByStatus) => {
            return `${pluralize(fixable, 'vulnerability', 'vulnerabilities')} with available fixes`;
        },
    },
    {
        status: 'Not fixable',
        Icon: MinusIcon,
        text: ({ total, fixable }: PlatformCVECountByStatus) => {
            return `${pluralize(total - fixable, 'vulnerability', 'vulnerabilities')} without fixes`;
        },
    },
] as const;

const statusHiddenText = {
    Fixable: 'Fixable hidden',
    'Not fixable': 'Not fixable hidden',
} as const;

export const platformCveCountByStatusFragment = gql`
    fragment PlatformCveCountByStatusFragment on PlatformCVECountByFixability {
        total
        fixable
    }
`;

export type PlatformCVECountByStatus = {
    total: number;
    fixable: number;
};

export type PlatformCvesByStatusSummaryCardProps = {
    data: PlatformCVECountByStatus;
    hiddenStatuses: Set<FixableStatus>;
};

function PlatformCvesByStatusSummaryCard({
    data,
    hiddenStatuses,
}: PlatformCvesByStatusSummaryCardProps) {
    return (
        <Card isCompact isFullHeight>
            <CardTitle>CVEs by status</CardTitle>
            <CardBody>
                <Flex direction={{ default: 'column' }}>
                    {statusDisplays.map(({ status, Icon, text }) => {
                        const isHidden = hiddenStatuses.has(status);
                        return (
                            <Flex
                                key={status}
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <Icon />
                                <Content
                                    component="p"
                                    style={{ color: isHidden ? disabledColor : 'inherit' }}
                                >
                                    {isHidden ? statusHiddenText[status] : text(data)}
                                </Content>
                            </Flex>
                        );
                    })}
                </Flex>
            </CardBody>
        </Card>
    );
}

export default PlatformCvesByStatusSummaryCard;
