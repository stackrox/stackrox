import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';
import isObject from 'lodash/isObject';

import DescriptionListItem from 'Components/DescriptionListItem';

type ObjectDescriptionListProps = {
    data: Record<string, any>;
};

function ObjectDescriptionList({ data }: ObjectDescriptionListProps): ReactElement {
    return (
        <DescriptionList isHorizontal>
            {Object.keys(data).map((key) => (
                <>
                    {data[key] !== undefined && data[key] !== null && (
                        <DescriptionListItem
                            term={key}
                            desc={
                                isObject(data[key]) ? (
                                    <ObjectDescriptionList data={data[key]} />
                                ) : (
                                    data[key]
                                )
                            }
                        />
                    )}
                </>
            ))}
        </DescriptionList>
    );
}

export default ObjectDescriptionList;
