import React, { ReactElement } from 'react';
import { List, ListItem, Text } from '@patternfly/react-core';

const listItemRegExp = /^- /;

type LineOrList = string | string[];

function splitHint(hint: string): LineOrList[] {
    const lineOrListArray: LineOrList[] = [];
    const listItemArray: string[] = [];

    function pushListItems() {
        if (listItemArray.length !== 0) {
            lineOrListArray.push([...listItemArray]);
            listItemArray.length = 0;
        }
    }

    hint.split('\n').forEach((line) => {
        if (listItemRegExp.test(line)) {
            listItemArray.push(line.slice(2));
        } else {
            pushListItems();
            lineOrListArray.push(line);
        }
    });
    pushListItems();

    return lineOrListArray;
}

export type AdministrationEventHintProps = {
    hint: string;
};

function AdministrationEventHint({ hint }: AdministrationEventHintProps): ReactElement {
    /* eslint-disable no-nested-ternary */
    /* eslint-disable react/no-array-index-key */
    return (
        <div>
            {splitHint(hint).map((lineOrList, lineOrListIndex) =>
                Array.isArray(lineOrList) ? (
                    <List key={lineOrListIndex}>
                        {lineOrList.map((listItem, listItemIndex) => (
                            <ListItem key={listItemIndex} className="pf-u-mt-0">
                                {listItem}
                            </ListItem>
                        ))}
                    </List>
                ) : lineOrList.length === 0 ? (
                    <br key={lineOrListIndex} />
                ) : (
                    <Text key={lineOrListIndex}>{lineOrList}</Text>
                )
            )}
        </div>
    );
    /* eslint-enable react/no-array-index-key */
    /* eslint-enable no-nested-ternary */
}

export default AdministrationEventHint;
