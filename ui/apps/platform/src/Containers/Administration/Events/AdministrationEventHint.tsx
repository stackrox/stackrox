import React, { ReactElement } from 'react';
import { List, ListItem, Text } from '@patternfly/react-core';

/*
 * Split hint string for conditional rendering in component:
 * Return adjacent lines which start with hyphen space as array for bulleted list.
 * Return other lines as string for br element if empty else paragraph.
 */

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
    // Remove default PatternFly margin-top for li + li to conserve vertical space.
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
                ) : lineOrList === '' ? (
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
