import { gql } from '@apollo/client';
import {
    Card,
    CardBody,
    CardTitle,
    Content,
    Flex,
    FlexItem,
    pluralize,
} from '@patternfly/react-core';

const statusDisplays = [
    {
        type: 'OpenShift CVE',
        field: 'openshift',
    },
    {
        type: 'Kubernetes CVE',
        field: 'kubernetes',
    },
    {
        type: 'Istio CVE',
        field: 'istio',
    },
] as const;

export const platformCveCountByTypeFragment = gql`
    fragment PlatformCveCountByTypeFragment on PlatformCVECountByType {
        kubernetes
        openshift
        istio
    }
`;

export type PlatformCVECountByType = {
    kubernetes: number;
    openshift: number;
    istio: number;
};

export type PlatformCvesByTypeSummaryCardProps = {
    data: PlatformCVECountByType;
};

function PlatformCvesByTypeSummaryCard({ data }: PlatformCvesByTypeSummaryCardProps) {
    return (
        <Card isCompact isFullHeight>
            <CardTitle>CVEs by type</CardTitle>
            <CardBody>
                <Flex direction={{ default: 'column' }}>
                    {statusDisplays.map(({ type, field }) => (
                        <FlexItem key={type} span={12}>
                            <Content component="p">{pluralize(data[field], type)}</Content>
                        </FlexItem>
                    ))}
                </Flex>
            </CardBody>
        </Card>
    );
}

export default PlatformCvesByTypeSummaryCard;
