import type { ReactElement } from 'react';
import { DescriptionList, DescriptionListTerm } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

type LeafValue = number | number[] | string | string[] | null | undefined;

// Enough for PortConfig in Deployment, but intentionally not recursive.
type ObjectValue = Record<string, LeafValue | Record<string, LeafValue>[]>;

type ObjectDescriptionListProps = {
    data: ObjectValue;
    className?: string;
};

function ObjectDescriptionList({ data, className }: ObjectDescriptionListProps): ReactElement {
    return (
        <DescriptionList isHorizontal className={className}>
            {Object.entries(data).map(([key, value]) => {
                if (typeof value === 'number' || typeof value === 'string') {
                    return (
                        <div key={key}>
                            <DescriptionListItem term={key} desc={value} />
                        </div>
                    );
                }

                if (Array.isArray(value)) {
                    if (value.length === 0) {
                        return null;
                    }

                    const [item0] = value;
                    return (
                        <div key={key}>
                            {typeof item0 !== 'object' && (
                                <DescriptionListTerm>{key}</DescriptionListTerm>
                            )}
                            <ArrayDescriptionList data={value} className="pf-v5-u-pl-md" />
                        </div>
                    );
                }

                if (value === null || value === undefined) {
                    return null;
                }

                return (
                    <div key={key}>
                        <ObjectDescriptionList data={value} className="pf-v5-u-pl-md" />
                    </div>
                );
            })}
        </DescriptionList>
    );
}

type ArrayValue = number[] | string[] | Record<string, LeafValue>[];

type ArrayDescriptionListProps = {
    data: ArrayValue;
    className?: string;
};

function ArrayDescriptionList({ data, className }: ArrayDescriptionListProps): ReactElement {
    return (
        <DescriptionList isHorizontal className={className}>
            {data.map((value, key) => (
                // eslint-disable-next-line react/no-array-index-key
                <div key={key}>
                    {typeof value === 'number' || typeof value === 'string' ? (
                        <DescriptionListItem term={key} desc={value} />
                    ) : (
                        <ObjectDescriptionList data={value} className="pf-v5-u-pl-md" />
                    )}
                </div>
            ))}
        </DescriptionList>
    );
}

export default ObjectDescriptionList;
