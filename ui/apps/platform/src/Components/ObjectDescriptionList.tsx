import React, { ReactElement } from 'react';
import { DescriptionList, DescriptionListTerm } from '@patternfly/react-core';
import isObject from 'lodash/isObject';
import isArray from 'lodash/isArray';

import DescriptionListItem from 'Components/DescriptionListItem';

type ObjectDescriptionListProps = {
    data: Record<string, any>;
    className?: string;
};

function ObjectDescriptionList({ data, className }: ObjectDescriptionListProps): ReactElement {
    const dataKeys = Object.keys(data).filter(
        (key) =>
            data[key] !== undefined &&
            data[key] !== null &&
            // filtering out empty arrays
            (isArray(data[key]) ? data[key].length > 0 : true)
    );
    return (
        <DescriptionList isHorizontal className={className}>
            {dataKeys.map((key) => (
                <div key={key}>
                    {isObject(data[key]) || isArray(data[key]) ? (
                        <>
                            {!isObject(data[key]) && (
                                <DescriptionListTerm>{key}</DescriptionListTerm>
                            )}
                            <ObjectDescriptionList data={data[key]} className="pf-u-pl-md" />
                        </>
                    ) : (
                        <DescriptionListItem term={key} desc={data[key]} />
                    )}
                </div>
            ))}
        </DescriptionList>
    );
}

export default ObjectDescriptionList;
